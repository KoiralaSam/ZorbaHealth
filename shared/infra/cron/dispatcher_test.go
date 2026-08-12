package cron

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRegisterRequiresFields(t *testing.T) {
	d := New(slog.Default())
	if err := d.Register(Job{Schedule: "* * * * *", Run: func(context.Context) error { return nil }}); err == nil {
		t.Fatal("expected error for missing name")
	}
	if err := d.Register(Job{Name: "x", Run: func(context.Context) error { return nil }}); err == nil {
		t.Fatal("expected error for missing schedule")
	}
	if err := d.Register(Job{Name: "x", Schedule: "* * * * *"}); err == nil {
		t.Fatal("expected error for missing run")
	}
}

func TestJobFiresOnSchedule(t *testing.T) {
	d := New(slog.Default())
	var runs atomic.Int32
	err := d.Register(Job{
		Name:     "tick",
		Schedule: "@every 1s",
		Timeout:  time.Second,
		Run: func(context.Context) error {
			runs.Add(1)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	d.Start()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = d.Stop(ctx)
	}()

	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if runs.Load() >= 2 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("expected at least 2 runs, got %d", runs.Load())
}

func TestSkipIfStillRunning(t *testing.T) {
	d := New(slog.Default())
	var started atomic.Int32
	var concurrent atomic.Int32
	var maxConcurrent atomic.Int32
	block := make(chan struct{})

	err := d.Register(Job{
		Name:     "slow",
		Schedule: "@every 1s",
		Timeout:  10 * time.Second,
		Run: func(ctx context.Context) error {
			started.Add(1)
			n := concurrent.Add(1)
			for {
				cur := maxConcurrent.Load()
				if n <= cur || maxConcurrent.CompareAndSwap(cur, n) {
					break
				}
			}
			select {
			case <-block:
			case <-ctx.Done():
			}
			concurrent.Add(-1)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	d.Start()
	defer func() {
		close(block)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = d.Stop(ctx)
	}()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if started.Load() >= 1 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if started.Load() < 1 {
		t.Fatal("job never started")
	}
	// Wait long enough for at least one more schedule tick while the first run is blocked.
	time.Sleep(1500 * time.Millisecond)

	if maxConcurrent.Load() > 1 {
		t.Fatalf("expected at most 1 concurrent run, got %d", maxConcurrent.Load())
	}
	if started.Load() != 1 {
		t.Fatalf("expected exactly 1 started run while blocked, got %d", started.Load())
	}
}

func TestJobErrorContained(t *testing.T) {
	d := New(slog.Default())
	var runs atomic.Int32
	err := d.Register(Job{
		Name:     "failing",
		Schedule: "@every 1s",
		Timeout:  time.Second,
		Run: func(context.Context) error {
			runs.Add(1)
			return errors.New("boom")
		},
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	d.Start()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = d.Stop(ctx)
	}()

	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if runs.Load() >= 2 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("expected job to keep running after errors, got %d runs", runs.Load())
}

func TestJobPanicContained(t *testing.T) {
	d := New(slog.Default())
	var runs atomic.Int32
	err := d.Register(Job{
		Name:     "panicking",
		Schedule: "@every 1s",
		Timeout:  time.Second,
		Run: func(context.Context) error {
			n := runs.Add(1)
			if n == 1 {
				panic("kaboom")
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	d.Start()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = d.Stop(ctx)
	}()

	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if runs.Load() >= 2 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("expected job to recover from panic and continue, got %d runs", runs.Load())
}

func TestGracefulStop(t *testing.T) {
	d := New(slog.Default())
	var wg sync.WaitGroup
	started := make(chan struct{})
	release := make(chan struct{})

	err := d.Register(Job{
		Name:     "blocking",
		Schedule: "@every 1s",
		Timeout:  5 * time.Second,
		Run: func(context.Context) error {
			wg.Add(1)
			defer wg.Done()
			select {
			case <-started:
			default:
				close(started)
			}
			<-release
			return nil
		},
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	d.Start()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("job did not start")
	}

	stopDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		stopDone <- d.Stop(ctx)
	}()

	time.Sleep(50 * time.Millisecond)
	close(release)

	select {
	case err := <-stopDone:
		if err != nil {
			t.Fatalf("stop: %v", err)
		}
	case <-time.After(4 * time.Second):
		t.Fatal("stop did not complete")
	}
	wg.Wait()
}

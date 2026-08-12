package cron

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/KoiralaSam/ZorbaHealth/shared/tracing"
)

const defaultTimeout = 55 * time.Second

// Job describes a named cron schedule and the work to run on each tick.
type Job struct {
	Name     string
	Schedule string
	Timeout  time.Duration
	Run      func(ctx context.Context) error
}

// Dispatcher schedules registered jobs with OpenTelemetry instrumentation.
type Dispatcher struct {
	cron   *cron.Cron
	logger *slog.Logger
	jobs   []Job
}

// New creates a Dispatcher. logger may be nil (falls back to slog.Default).
func New(logger *slog.Logger) *Dispatcher {
	if logger == nil {
		logger = slog.Default()
	}
	return &Dispatcher{
		cron: cron.New(
			cron.WithChain(
				cron.Recover(cron.PrintfLogger(slog.NewLogLogger(logger.Handler(), slog.LevelError))),
				cron.SkipIfStillRunning(cron.PrintfLogger(slog.NewLogLogger(logger.Handler(), slog.LevelInfo))),
			),
		),
		logger: logger,
	}
}

// Register adds a job. Must be called before Start.
func (d *Dispatcher) Register(job Job) error {
	if job.Name == "" {
		return fmt.Errorf("cron job name is required")
	}
	if job.Schedule == "" {
		return fmt.Errorf("cron job %q schedule is required", job.Name)
	}
	if job.Run == nil {
		return fmt.Errorf("cron job %q run function is required", job.Name)
	}
	if job.Timeout <= 0 {
		job.Timeout = defaultTimeout
	}

	jobCopy := job
	_, err := d.cron.AddFunc(job.Schedule, func() {
		d.runJob(jobCopy)
	})
	if err != nil {
		return fmt.Errorf("register cron job %q: %w", job.Name, err)
	}
	d.jobs = append(d.jobs, jobCopy)
	d.logger.Info("cron job registered", "job", job.Name, "schedule", job.Schedule)
	return nil
}

// Start begins executing registered jobs on their schedules.
func (d *Dispatcher) Start() {
	d.logger.Info("cron dispatcher starting", "jobs", len(d.jobs))
	d.cron.Start()
}

// Stop gracefully stops the dispatcher and waits for in-flight runs
// (or until ctx is cancelled).
func (d *Dispatcher) Stop(ctx context.Context) error {
	stopCtx := d.cron.Stop()
	select {
	case <-stopCtx.Done():
		d.logger.Info("cron dispatcher stopped")
		return nil
	case <-ctx.Done():
		d.logger.Warn("cron dispatcher stop timed out waiting for in-flight jobs")
		return ctx.Err()
	}
}

func (d *Dispatcher) runJob(job Job) {
	ctx := context.Background()
	tracer := tracing.GetTracer("shared/infra/cron")
	ctx, span := tracer.Start(ctx, "cron.job.run")
	defer span.End()

	span.SetAttributes(
		attribute.String("cron.job.name", job.Name),
		attribute.String("cron.job.schedule", job.Schedule),
	)

	start := time.Now()
	logger := d.logger.With("job", job.Name, "schedule", job.Schedule)
	logger.InfoContext(ctx, "cron job started")

	runCtx, cancel := context.WithTimeout(ctx, job.Timeout)
	defer cancel()

	err := func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic: %v", r)
			}
		}()
		return job.Run(runCtx)
	}()

	duration := time.Since(start)
	span.SetAttributes(attribute.Int64("cron.job.duration_ms", duration.Milliseconds()))

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.ErrorContext(ctx, "cron job failed", "error", err, "duration_ms", duration.Milliseconds())
		return
	}

	span.SetStatus(codes.Ok, "")
	logger.InfoContext(ctx, "cron job completed", "duration_ms", duration.Milliseconds())
}

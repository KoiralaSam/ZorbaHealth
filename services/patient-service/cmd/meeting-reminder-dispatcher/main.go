package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	livekitadapter "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/adapters/secondary/external/livekit"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/adapters/secondary/messaging/rabbitmq"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/adapters/secondary/repositories/postgres"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/services"
	"github.com/KoiralaSam/ZorbaHealth/shared/db"
	"github.com/KoiralaSam/ZorbaHealth/shared/env"
	"github.com/KoiralaSam/ZorbaHealth/shared/events"
	"github.com/KoiralaSam/ZorbaHealth/shared/messaging"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := db.InitDB(ctx, env.GetString("DATABASE_URL", "")); err != nil {
		log.Fatalf("database init failed: %v", err)
	}
	defer db.GetDB().Close()

	ready, err := meetingReminderSchemaReady(ctx)
	if err != nil {
		log.Fatalf("meeting reminder schema readiness check failed: %v", err)
	}
	if !ready {
		log.Printf("meeting reminder schema is not migrated yet; skipping dispatch")
		return
	}

	rmq, err := messaging.NewRabbitMQ(env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/"), events.PatientExchange, events.PatientPublisherQueueBindings)
	if err != nil {
		log.Fatalf("rabbitmq init failed: %v", err)
	}
	defer rmq.Close()

	meetingRepo := postgres.NewMeetingRepository(db.GetDB())
	patientRepo := postgres.NewPatientRepository(db.GetDB())
	publisher := rabbitmq.NewSchedulingPublisher(rmq)
	livekit := livekitadapter.NewClient()
	svc := services.NewSchedulingService(meetingRepo, patientRepo, nil, livekit, publisher, nil)

	limit := int32(env.GetInt("MEETING_REMINDER_DISPATCH_LIMIT", 25))
	loopSeconds := env.GetInt("MEETING_REMINDER_DISPATCH_LOOP_SECONDS", 0)

	for {
		if err := runOnce(ctx, svc, limit); err != nil {
			if loopSeconds <= 0 {
				log.Fatalf("meeting reminder dispatch failed: %v", err)
			}
			log.Printf("meeting reminder dispatch failed: %v", err)
		}
		if loopSeconds <= 0 {
			return
		}
		timer := time.NewTimer(time.Duration(loopSeconds) * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func runOnce(ctx context.Context, svc *services.SchedulingService, limit int32) error {
	runCtx, cancel := context.WithTimeout(ctx, 55*time.Second)
	defer cancel()
	sent, err := svc.DispatchDueMeetingReminders(runCtx, limit)
	if err != nil {
		return err
	}
	log.Printf("meeting reminder dispatcher sent %d reminder(s)", sent)
	return nil
}

func meetingReminderSchemaReady(ctx context.Context) (bool, error) {
	var ready bool
	err := db.GetDB().QueryRow(ctx, `
		SELECT to_regclass('public.scheduled_meetings') IS NOT NULL
		   AND EXISTS (
		     SELECT 1 FROM information_schema.columns
		     WHERE table_name = 'scheduled_meetings' AND column_name = 'reminder_sent_at'
		   )
	`).Scan(&ready)
	return ready, err
}

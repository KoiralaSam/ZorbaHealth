package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	authsvc "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/adapters/secondary/external/auth"
	livekitadapter "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/adapters/secondary/external/livekit"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/adapters/secondary/messaging/rabbitmq"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/adapters/secondary/repositories/postgres"
	redisrepo "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/adapters/secondary/repositories/redis"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/services"
	"github.com/KoiralaSam/ZorbaHealth/shared/db"
	"github.com/KoiralaSam/ZorbaHealth/shared/env"
	"github.com/KoiralaSam/ZorbaHealth/shared/events"
	"github.com/KoiralaSam/ZorbaHealth/shared/grpcclient"
	sharedcron "github.com/KoiralaSam/ZorbaHealth/shared/infra/cron"
	"github.com/KoiralaSam/ZorbaHealth/shared/logging"
	"github.com/KoiralaSam/ZorbaHealth/shared/messaging"
	auditpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/audit"
	"github.com/KoiralaSam/ZorbaHealth/shared/tracing"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	environment := env.GetString("ENVIRONMENT", "development")
	jaegerEndpoint := env.GetString("JAEGER_ENDPOINT", "http://localhost:4318/v1/traces")

	tracerShutdown, err := tracing.InitTracer(tracing.Config{
		ServiceName:    "cron-dispatcher",
		Environment:    environment,
		JaegerEndpoint: jaegerEndpoint,
	})
	if err != nil {
		log.Fatalf("tracer init failed: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tracerShutdown(shutdownCtx)
	}()

	logger, loggerShutdown, err := logging.InitLogger(logging.Config{
		ServiceName:  "cron-dispatcher",
		Environment:  environment,
		OTLPEndpoint: env.GetString("OTEL_EXPORTER_OTLP_LOGS_ENDPOINT", ""),
	})
	if err != nil {
		log.Fatalf("logger init failed: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = loggerShutdown(shutdownCtx)
	}()
	slog.SetDefault(logger)

	if err := db.InitDB(ctx, env.GetString("DATABASE_URL", "")); err != nil {
		logger.Error("database init failed", "error", err)
		os.Exit(1)
	}
	defer db.GetDB().Close()

	patientRepo := postgres.NewPatientRepository(db.GetDB())
	authRepo, err := authsvc.NewAuthRepository(env.GetString("AUTH_SERVICE_GRPC_ADDR", "auth-service:9092"))
	if err != nil {
		logger.Error("auth repository init failed", "error", err)
		os.Exit(1)
	}
	if closer, ok := authRepo.(authsvc.AuthRepositoryWithClose); ok {
		defer closer.Close()
	}
	pendingRegRepo, err := redisrepo.NewPendingRegistrationRepository()
	if err != nil {
		logger.Error("redis repository init failed", "error", err)
		os.Exit(1)
	}

	var auditClient auditpb.AuditServiceClient
	auditConn, err := grpcclient.DialService(env.GetString("AUDIT_SERVICE_GRPC_ADDR", "audit-service:50058"))
	if err != nil {
		logger.Warn("audit-service dial failed; consent checks will fail closed unless bypass is enabled", "error", err)
	} else {
		defer auditConn.Close()
		auditClient = auditpb.NewAuditServiceClient(auditConn)
	}

	patientSvc := services.NewPatientService(
		patientRepo,
		authRepo,
		pendingRegRepo,
		nil,
		auditClient,
		livekitadapter.NewWelfareCheckCallProvider(),
	)

	rmq, err := messaging.NewRabbitMQ(
		env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/"),
		events.PatientExchange,
		events.PatientPublisherQueueBindings,
	)
	if err != nil {
		logger.Error("rabbitmq init failed", "error", err)
		os.Exit(1)
	}
	defer rmq.Close()

	meetingRepo := postgres.NewMeetingRepository(db.GetDB())
	schedulingSvc := services.NewSchedulingService(
		meetingRepo,
		patientRepo,
		nil,
		livekitadapter.NewClient(),
		rabbitmq.NewSchedulingPublisher(rmq),
		nil,
	)

	welfareLimit := int32(env.GetInt("WELFARE_CHECK_DISPATCH_LIMIT", 25))
	meetingLimit := int32(env.GetInt("MEETING_REMINDER_DISPATCH_LIMIT", 25))
	welfareCron := env.GetString("WELFARE_CHECK_DISPATCH_CRON", "* * * * *")
	meetingCron := env.GetString("MEETING_REMINDER_DISPATCH_CRON", "* * * * *")

	dispatcher := sharedcron.New(logger)
	if err := dispatcher.Register(sharedcron.Job{
		Name:     "welfare-check-dispatch",
		Schedule: welfareCron,
		// SIP WaitUntilAnswered can ring up to ~45s; keep headroom for auth/consent.
		Timeout: 90 * time.Second,
		Run: func(runCtx context.Context) error {
			return runWelfareCheckDispatch(runCtx, logger, patientSvc, welfareLimit)
		},
	}); err != nil {
		logger.Error("register welfare-check job failed", "error", err)
		os.Exit(1)
	}
	if err := dispatcher.Register(sharedcron.Job{
		Name:     "meeting-reminder-dispatch",
		Schedule: meetingCron,
		Timeout:  55 * time.Second,
		Run: func(runCtx context.Context) error {
			return runMeetingReminderDispatch(runCtx, logger, schedulingSvc, meetingLimit)
		},
	}); err != nil {
		logger.Error("register meeting-reminder job failed", "error", err)
		os.Exit(1)
	}

	dispatcher.Start()
	logger.Info("cron dispatcher running")

	<-ctx.Done()
	logger.Info("shutdown signal received")
	stopCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := dispatcher.Stop(stopCtx); err != nil {
		logger.Error("cron dispatcher stop failed", "error", err)
		os.Exit(1)
	}
}

func runWelfareCheckDispatch(ctx context.Context, logger *slog.Logger, svc *services.PatientService, limit int32) error {
	ready, err := welfareCheckSchemaReady(ctx)
	if err != nil {
		return err
	}
	if !ready {
		logger.Info("welfare check schema is not migrated yet; skipping dispatch")
		return nil
	}
	results, err := svc.DispatchDueWelfareChecks(ctx, limit)
	if err != nil {
		return err
	}
	logger.Info("welfare check dispatcher processed runs", "count", len(results))
	return nil
}

func runMeetingReminderDispatch(ctx context.Context, logger *slog.Logger, svc *services.SchedulingService, limit int32) error {
	ready, err := meetingReminderSchemaReady(ctx)
	if err != nil {
		return err
	}
	if !ready {
		logger.Info("meeting reminder schema is not migrated yet; skipping dispatch")
		return nil
	}
	sent, err := svc.DispatchDueMeetingReminders(ctx, limit)
	if err != nil {
		return err
	}
	logger.Info("meeting reminder dispatcher sent reminders", "count", sent)
	return nil
}

func welfareCheckSchemaReady(ctx context.Context) (bool, error) {
	var ready bool
	err := db.GetDB().QueryRow(ctx, `
		SELECT to_regclass('public.welfare_check_requests') IS NOT NULL
		   AND to_regclass('public.welfare_check_runs') IS NOT NULL
	`).Scan(&ready)
	return ready, err
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

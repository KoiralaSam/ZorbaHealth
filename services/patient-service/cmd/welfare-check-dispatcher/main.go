package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	authsvc "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/adapters/secondary/external/auth"
	livekitadapter "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/adapters/secondary/external/livekit"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/adapters/secondary/repositories/postgres"
	redisrepo "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/adapters/secondary/repositories/redis"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/services"
	"github.com/KoiralaSam/ZorbaHealth/shared/db"
	"github.com/KoiralaSam/ZorbaHealth/shared/env"
	"github.com/KoiralaSam/ZorbaHealth/shared/grpcclient"
	auditpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/audit"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := db.InitDB(ctx, env.GetString("DATABASE_URL", "")); err != nil {
		log.Fatalf("database init failed: %v", err)
	}
	defer db.GetDB().Close()

	ready, err := welfareCheckSchemaReady(ctx)
	if err != nil {
		log.Fatalf("welfare check schema readiness check failed: %v", err)
	}
	if !ready {
		log.Printf("welfare check schema is not migrated yet; skipping dispatch")
		return
	}

	repo := postgres.NewPatientRepository(db.GetDB())
	authRepo, err := authsvc.NewAuthRepository(env.GetString("AUTH_SERVICE_GRPC_ADDR", "auth-service:9092"))
	if err != nil {
		log.Fatalf("auth repository init failed: %v", err)
	}
	if closer, ok := authRepo.(authsvc.AuthRepositoryWithClose); ok {
		defer closer.Close()
	}
	pendingRegRepo, err := redisrepo.NewPendingRegistrationRepository()
	if err != nil {
		log.Fatalf("redis repository init failed: %v", err)
	}

	var auditClient auditpb.AuditServiceClient
	auditConn, err := grpcclient.Dial(env.GetString("AUDIT_SERVICE_GRPC_ADDR", "audit-service:50058"))
	if err != nil {
		log.Printf("audit-service dial failed: %v (consent checks will fail closed unless bypass is enabled)", err)
	} else {
		defer auditConn.Close()
		auditClient = auditpb.NewAuditServiceClient(auditConn)
	}

	svc := services.NewPatientService(repo, authRepo, pendingRegRepo, nil, auditClient, livekitadapter.NewWelfareCheckCallProvider())
	limit := int32(env.GetInt("WELFARE_CHECK_DISPATCH_LIMIT", 25))
	loopSeconds := env.GetInt("WELFARE_CHECK_DISPATCH_LOOP_SECONDS", 0)

	for {
		if err := runOnce(ctx, svc, limit); err != nil {
			if loopSeconds <= 0 {
				log.Fatalf("welfare check dispatch failed: %v", err)
			}
			log.Printf("welfare check dispatch failed: %v", err)
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

func runOnce(ctx context.Context, svc *services.PatientService, limit int32) error {
	runCtx, cancel := context.WithTimeout(ctx, 55*time.Second)
	defer cancel()
	results, err := svc.DispatchDueWelfareChecks(runCtx, limit)
	if err != nil {
		return err
	}
	log.Printf("welfare check dispatcher processed %d successful run(s)", len(results))
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

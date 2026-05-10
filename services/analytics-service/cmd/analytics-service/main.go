package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpchandlers "github.com/KoiralaSam/ZorbaHealth/services/analytics-service/internal/adapters/primary/grpc/handlers"
	postgresrepos "github.com/KoiralaSam/ZorbaHealth/services/analytics-service/internal/adapters/secondary/repositories/postgres"
	"github.com/KoiralaSam/ZorbaHealth/services/analytics-service/internal/core/services"
	"github.com/jackc/pgx/v5/pgxpool"
)

func grpcListenAddr(addr string, defaultPort string) string {
	if addr == "" {
		return ":" + defaultPort
	}
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return ":" + defaultPort
	}
	return ":" + port
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	db, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer db.Close()

	hospitalRepo := postgresrepos.NewHospitalAnalyticsRepository(db)
	platformRepo := postgresrepos.NewPlatformAnalyticsRepository(db)
	patientRepo := postgresrepos.NewPatientAnalyticsRepository(db)

	refresher := postgresrepos.NewViewRefresher(db, time.Hour)
	refresher.Start(ctx)

	svc := services.NewAnalyticsService(hospitalRepo, platformRepo, patientRepo)

	grpcAddr := grpcListenAddr(os.Getenv("ANALYTICS_SERVICE_GRPC_ADDR"), "50054")
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	defer lis.Close()

	grpcServer := grpchandlers.NewServer(db, svc)

	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
	}()

	log.Printf("analytics-service listening on %s", grpcAddr)
	log.Fatal(grpcServer.Serve(lis))
}

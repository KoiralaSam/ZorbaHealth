package main

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/KoiralaSam/ZorbaHealth/services/mcp-server/tools"
	sharedenv "github.com/KoiralaSam/ZorbaHealth/shared/env"
	"github.com/KoiralaSam/ZorbaHealth/shared/grpcclient"
	analyticspb "github.com/KoiralaSam/ZorbaHealth/shared/proto/analytics"
	healthpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/health_records"
	locpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/location"
	transpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/translation"
)

func main() {
	ctx := context.Background()

	db, err := pgxpool.New(ctx, sharedenv.GetString("DATABASE_URL", ""))
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}

	healthConn, err := grpcclient.Dial(sharedenv.GetString("HEALTH_RECORDS_SERVICE_GRPC_ADDR", "health-records-service:50054"))
	if err != nil {
		log.Fatalf("health-records-service dial: %v", err)
	}
	transConn, err := grpcclient.Dial(sharedenv.GetString("TRANSLATION_SERVICE_GRPC_ADDR", "translation-service:50057"))
	if err != nil {
		log.Fatalf("translation-service dial: %v", err)
	}
	locConn, err := grpcclient.Dial(sharedenv.GetString("LOCATION_SERVICE_GRPC_ADDR", "location-service:50051"))
	if err != nil {
		log.Fatalf("location-service dial: %v", err)
	}
	analyticsConn, err := grpcclient.Dial(sharedenv.GetString("ANALYTICS_SERVICE_GRPC_ADDR", "analytics-service:50054"))
	if err != nil {
		log.Fatalf("analytics-service dial: %v", err)
	}

	defer healthConn.Close()
	defer transConn.Close()
	defer locConn.Close()
	defer analyticsConn.Close()

	healthClient := healthpb.NewHealthRecordServiceClient(healthConn)
	transClient := transpb.NewTranslationServiceClient(transConn)
	locClient := locpb.NewLocationServiceClient(locConn)
	analyticsClient := analyticspb.NewAnalyticsServiceClient(analyticsConn)

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "zorba-health-mcp",
		Version: "1.0.0",
	}, nil)

	tools.RegisterTranslate(server, db, transClient)
	tools.RegisterSearchHealthRecords(server, db, healthClient)
	tools.RegisterSummarizePatientRecord(server, db, healthClient)
	tools.RegisterGetLocation(server, db, locClient)
	tools.RegisterFindNearestHospital(server, db, locClient)
	tools.RegisterGetHospitalAnalytics(server, db, analyticsClient)

	log.Println("MCP server starting on stdio...")
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/KoiralaSam/ZorbaHealth/services/mcp-server/tools"
	sharedenv "github.com/KoiralaSam/ZorbaHealth/shared/env"
	"github.com/KoiralaSam/ZorbaHealth/shared/events"
	"github.com/KoiralaSam/ZorbaHealth/shared/grpcclient"
	"github.com/KoiralaSam/ZorbaHealth/shared/messaging"
	auditpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/audit"
	analyticspb "github.com/KoiralaSam/ZorbaHealth/shared/proto/analytics"
	healthpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/health_records"
	locpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/location"
	regpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/patient/registration_verification"
	patientportalpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/patientportal"
	schedpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/patient/scheduling"
	transpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/translation"
	"github.com/KoiralaSam/ZorbaHealth/shared/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	tracerCfg := tracing.Config{
		ServiceName:    "mcp-server",
		Environment:    sharedenv.GetString("ENVIRONMENT", "development"),
		JaegerEndpoint: sharedenv.GetString("JAEGER_ENDPOINT", "http://localhost:4318/v1/traces"),
	}
	sh, err := tracing.InitTracer(tracerCfg)
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer sh(ctx)
	defer cancel()

	db, err := pgxpool.New(ctx, sharedenv.GetString("DATABASE_URL", ""))
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	rabbitURI := sharedenv.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")
	rabbitmq, err := messaging.NewRabbitMQ(rabbitURI, events.EscalationExchange, nil)
	if err != nil {
		log.Fatalf("rabbitmq escalation: %v", err)
	}
	callsRabbitMQ, err := messaging.NewRabbitMQ(rabbitURI, events.CallsExchange, events.LocationServiceCallsQueueBindings)
	if err != nil {
		log.Fatalf("rabbitmq calls: %v", err)
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
	patientConn, err := grpcclient.Dial(sharedenv.GetString("PATIENT_SERVICE_GRPC_ADDR", "patient-service:9093"))
	if err != nil {
		log.Fatalf("patient-service dial: %v", err)
	}
	auditConn, err := grpcclient.Dial(sharedenv.GetString("AUDIT_SERVICE_GRPC_ADDR", "audit-service:50058"))
	if err != nil {
		log.Fatalf("audit-service dial: %v", err)
	}

	defer healthConn.Close()
	defer transConn.Close()
	defer locConn.Close()
	defer analyticsConn.Close()
	defer patientConn.Close()
	defer auditConn.Close()
	defer rabbitmq.Close()
	defer callsRabbitMQ.Close()

	healthClient := healthpb.NewHealthRecordServiceClient(healthConn)
	transClient := transpb.NewTranslationServiceClient(transConn)
	locClient := locpb.NewLocationServiceClient(locConn)
	analyticsClient := analyticspb.NewAnalyticsServiceClient(analyticsConn)
	patientClient := regpb.NewRegistrationVerificationServiceClient(patientConn)
	patientPortalClient := patientportalpb.NewPatientPortalServiceClient(patientConn)
	schedulingClient := schedpb.NewSchedulingServiceClient(patientConn)
	auditClient := auditpb.NewAuditServiceClient(auditConn)
	tools.ConfigureAuditClient(auditClient)

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "zorba-health-mcp",
		Version: "1.0.0",
	}, nil)

	tools.RegisterTranslate(server, db, transClient)
	tools.RegisterSearchHealthRecords(server, db, healthClient)
	tools.RegisterAnswerHealthQuestion(server, db, healthClient)
	tools.RegisterSummarizePatientRecord(server, db, healthClient)
	tools.RegisterGetLocation(server, db, locClient)
	tools.RegisterFindNearestHospital(server, db, locClient)
	tools.RegisterGetHospitalAnalytics(server, db, analyticsClient)
	tools.RegisterLogEscalation(server, db, rabbitmq)
	tools.RegisterLookupPatientByPhone(server, db, patientClient)
	tools.RegisterStartExistingPhoneVerification(server, db, patientClient)
	tools.RegisterVerifyExistingPhoneOTP(server, db, patientClient)
	tools.RegisterConsumeVoiceVerification(server, db, patientClient)
	tools.RegisterNotifyCallLifecycle(server, db, callsRabbitMQ)
	tools.RegisterUpdateWelfareRunStatus(server, db, patientPortalClient)
	tools.RegisterListPatientHospitals(server, db)
	tools.RegisterListSchedulableStaff(server, db, schedulingClient)
	tools.RegisterScheduleHealthStaffMeeting(server, db, schedulingClient)

	if strings.EqualFold(sharedenv.GetString("MCP_TRANSPORT", "stdio"), "http") {
		addr := sharedenv.GetString("MCP_HTTP_ADDR", ":8092")
		handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
			return server
		}, &mcp.StreamableHTTPOptions{
			Stateless:    true,
			JSONResponse: true,
		})
		log.Printf("MCP server starting on streamable HTTP at %s", addr)
		log.Fatal(http.ListenAndServe(addr, otelhttp.NewHandler(handler, "mcp-server")))
	}

	log.Println("MCP server starting on stdio...")
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}

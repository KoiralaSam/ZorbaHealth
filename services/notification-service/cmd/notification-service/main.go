package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	rmqconsumer "github.com/KoiralaSam/ZorbaHealth/services/notification-service/internal/adapters/primary/events/rabbitmq"
	httpadapter "github.com/KoiralaSam/ZorbaHealth/services/notification-service/internal/adapters/primary/http"
	"github.com/KoiralaSam/ZorbaHealth/services/notification-service/internal/adapters/secondary/email"
	"github.com/KoiralaSam/ZorbaHealth/services/notification-service/internal/adapters/secondary/sms/voipms"
	"github.com/KoiralaSam/ZorbaHealth/services/notification-service/internal/core/services"
	"github.com/KoiralaSam/ZorbaHealth/shared/env"
	"github.com/KoiralaSam/ZorbaHealth/shared/events"
	"github.com/KoiralaSam/ZorbaHealth/shared/messaging"
	"github.com/KoiralaSam/ZorbaHealth/shared/tracing"
)

func main() {
	log.Println("Starting notification service")
	tracerCfg := tracing.Config{
		ServiceName:    "notification-service",
		Environment:    env.GetString("ENVIRONMENT", "development"),
		JaegerEndpoint: env.GetString("JAEGER_ENDPOINT", "http://localhost:4318/v1/traces"),
	}
	sh, err := tracing.InitTracer(tracerCfg)
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer sh(ctx)
	defer cancel()

	httpAddr := env.GetString("HTTP_ADDR", ":3000")
	webhookAPIKey := env.GetString("VOIPMS_API_KEY", "")
	voipmsDID := env.GetString("VOIPMS_DID", "")
	voipmsUsername := env.GetString("VOIPMS_API_USERNAME", "")
	voipmsPassword := env.GetString("VOIPMS_API_PASSWORD", "")
	voipmsBaseURL := env.GetString("VOIPMS_API_BASEURL", "")

	rabbitmq, err := messaging.NewRabbitMQ(env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/"), events.PatientExchange, events.NotificationServicePatientQueueBindings)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
		return
	}

	defer rabbitmq.Close()
	log.Println("Starting RabbitMQ connection")

	smsSender := voipms.NewSender(voipmsBaseURL, voipmsUsername, voipmsPassword, voipmsDID)

	mailtrapAPIToken := env.GetString("MAILTRAP_API_TOKEN", "")
	if mailtrapAPIToken == "" {
		log.Fatalf("MAILTRAP_API_TOKEN is required")
	}
	fromEmail := env.GetString("MAILTRAP_FROM_EMAIL", "")
	if fromEmail == "" {
		log.Fatalf("MAILTRAP_FROM_EMAIL is required")
	}
	fromName := env.GetString("MAILTRAP_FROM_NAME", "ZorbaHealth")
	mailtrapSendURL := env.GetString("MAILTRAP_SEND_URL", "")
	mailtrapMirrorRecipient := env.GetString("MAILTRAP_MIRROR_RECIPIENT", "")

	publicWebBaseURL := env.GetString("PUBLIC_WEB_BASE_URL", "")

	emailSender := email.NewMailtrapSender(mailtrapAPIToken, fromEmail, fromName, mailtrapSendURL, mailtrapMirrorRecipient)
	notificationSvc := services.NewNotificationService(emailSender, smsSender, nil, publicWebBaseURL)

	//webhook used by VoIP.ms to send SMS to the service
	httpServer := httpadapter.NewServer(httpAddr, webhookAPIKey, notificationSvc)
	go httpServer.Run()

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		cancel()
	}()

	//primary adapter to use the service
	patientConsumer := rmqconsumer.NewPatientConsumer(rabbitmq, notificationSvc)
	emergencyConsumer := rmqconsumer.NewEmergencyConsumer(rabbitmq, notificationSvc)
	go func() {
		if err := patientConsumer.Listen(); err != nil {
			log.Printf("Failed to listen for patient messages: %v", err)
			cancel()
		}
	}()
	go func() {
		if err := emergencyConsumer.Listen(); err != nil {
			log.Printf("Failed to listen for emergency escalation messages: %v", err)
			cancel()
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down notification service")
}

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
	"github.com/KoiralaSam/ZorbaHealth/shared/messaging"
)

func main() {
	log.Println("Starting notification service")

	httpAddr := env.GetString("HTTP_ADDR", ":3000")
	webhookAPIKey := env.GetString("VOIPMS_API_KEY", "")
	voipmsDID := env.GetString("VOIPMS_DID", "")
	voipmsUsername := env.GetString("VOIPMS_API_USERNAME", "")
	voipmsPassword := env.GetString("VOIPMS_API_PASSWORD", "")
	voipmsBaseURL := env.GetString("VOIPMS_API_BASEURL", "")

	rabbitmq, err := messaging.NewRabbitMQ(env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/"))
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
		return
	}

	defer rabbitmq.Close()
	log.Println("Starting RabbitMQ connection")

	smsSender := voipms.NewSender(voipmsBaseURL, voipmsUsername, voipmsPassword, voipmsDID)

	sendGridAPIKey := env.GetString("SENDGRID_API_KEY", "")
	if sendGridAPIKey == "" {
		log.Fatalf("SENDGRID_API_KEY is required")
	}
	fromEmail := env.GetString("SENDGRID_FROM_EMAIL", "")
	if fromEmail == "" {
		log.Fatalf("SENDGRID_FROM_EMAIL is required")
	}
	fromName := env.GetString("SENDGRID_FROM_NAME", "ZorbaHealth")

	publicWebBaseURL := env.GetString("PUBLIC_WEB_BASE_URL", "")

	emailSender := email.NewSendGridSender(sendGridAPIKey, fromEmail, fromName)
	notificationSvc := services.NewNotificationService(emailSender, smsSender, nil, publicWebBaseURL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
	go func() {
		if err := patientConsumer.Listen(); err != nil {
			log.Printf("Failed to listen for patient messages: %v", err)
			cancel()
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down notification service")
}

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	rmqconsumer "github.com/KoiralaSam/ZorbaHealth/services/notification-service/internal/adapters/primary/events/rabbitmq"
	"github.com/KoiralaSam/ZorbaHealth/services/notification-service/internal/adapters/secondary/email"
	"github.com/KoiralaSam/ZorbaHealth/services/notification-service/internal/core/services"
	"github.com/KoiralaSam/ZorbaHealth/shared/env"
	"github.com/KoiralaSam/ZorbaHealth/shared/messaging"
)

func main() {
	rabbitmq, err := messaging.NewRabbitMQ(env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/"))
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rabbitmq.Close()
	log.Println("RabbitMQ connected")

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
	notificationSvc := services.NewNotificationService(emailSender, publicWebBaseURL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		cancel()
	}()

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

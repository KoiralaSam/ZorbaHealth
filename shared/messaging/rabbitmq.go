package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/KoiralaSam/ZorbaHealth/shared/contracts"
	"github.com/KoiralaSam/ZorbaHealth/shared/events"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	conn    *amqp.Connection
	Channel *amqp.Channel
}

// NewRabbitMQ connects and declares the given topic exchange plus each queue and its routing-key bindings.
// Pass exchange name and bindings from shared/events (per producer/consumer).
func NewRabbitMQ(uri string, exchange string, queueBindings []events.QueueBinding) (*RabbitMQ, error) {
	conn, err := amqp.Dial(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open a channel: %v", err)
	}
	rmq := &RabbitMQ{conn: conn, Channel: ch}

	if err := rmq.setupExchangesAndQueues(exchange, queueBindings); err != nil {
		rmq.Close()
		return nil, fmt.Errorf("failed to setup exchanges and queues: %v", err)
	}

	return rmq, nil
}

func (r *RabbitMQ) setupExchangesAndQueues(exchange string, bindings []events.QueueBinding) error {
	err := r.Channel.ExchangeDeclare(
		exchange, // name
		"topic",  // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %s: %v", exchange, err)
	}

	for _, b := range bindings {
		if err := r.declareAndBindQueue(b.QueueName, b.RoutingKeys, exchange); err != nil {
			return fmt.Errorf("failed to declare and bind queue %q: %v", b.QueueName, err)
		}
	}

	return nil
}

func (r *RabbitMQ) declareAndBindQueue(queueName string, messageTypes []string, exhange string) error {
	q, err := r.Channel.QueueDeclare(
		queueName, //name of the queue
		true,      //durable
		false,     //delete when unused
		false,     //exclusive
		false,     //no-wait
		nil,       //arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %v", err)
	}
	for _, msg := range messageTypes {
		err = r.Channel.QueueBind(
			q.Name,  // queue name
			msg,     // routing key
			exhange, // exchange
			false,
			nil,
		)
		if err != nil {
			return fmt.Errorf("failed to bind queue: %v", err)
		}
	}
	return nil
}

type MessageHandler func(ctx context.Context, message amqp.Delivery) error

func (r *RabbitMQ) ConsumeMessages(queueName string, handler MessageHandler) error {
	// Set prefetch count to 1 for fair dispatch
	// This tells RabbitMQ not to give more than one message to a service at a time.
	// The worker will only get the next message after it has acknowledged the previous one.
	err := r.Channel.Qos(
		1,     // prefetchCount: Limit to 1 unacknowledged message per consumer
		0,     // prefetchSize: No specific limit on message size
		false, // global: Apply prefetchCount to each consumer individually
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %v", err)
	}

	msgs, err := r.Channel.Consume(
		queueName, // queue
		"",        // consumer
		false,     // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return err
	}

	ctx := context.Background()

	go func() {
		for msg := range msgs {
			log.Printf("Received a message: %s", msg.Body)

			if err := handler(ctx, msg); err != nil {
				log.Printf("failed to handle message: %v. Message Body: %s", err, msg.Body)

				// Requeue by default so transient failures can be retried.
				if nackErr := msg.Nack(false, true); nackErr != nil {
					log.Printf("failed to nack message (will likely be redelivered on consumer restart): %v", nackErr)
				}

				//continue to the next message
				continue
			}
			//only ack the message if the handler succeeds
			if ackErr := msg.Ack(false); ackErr != nil {
				log.Printf("failed to ack message: %v. Message Body: %s", ackErr, msg.Body)
			}
		}
	}()
	return nil

}

func (r *RabbitMQ) PublishMessage(ctx context.Context, exhange string, routingKey string, messagage contracts.AmqpMessage) error {
	log.Printf("Publishing message to %s with routing key %s", exhange, routingKey)

	jsonMsg, err := json.Marshal(messagage)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}
	return r.Channel.PublishWithContext(ctx,
		exhange,    // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType:  "text/plain",
			Body:         jsonMsg,
			DeliveryMode: amqp.Persistent,
		})
}

func (r *RabbitMQ) Close() {
	if r.conn != nil {
		r.conn.Close()
	}
	if r.Channel != nil {
		r.Channel.Close()
	}
}

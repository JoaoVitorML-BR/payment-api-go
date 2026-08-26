// payment-consumer\worker\runner.go
package worker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/JoaoVitorML-BR/payment-api-go/payment-consumer/internal/config"
	"github.com/JoaoVitorML-BR/payment-api-go/payment-consumer/internal/infra/database/bridge"
	"github.com/JoaoVitorML-BR/payment-api-go/payment-consumer/internal/infra/paymentgateway"
	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func connectRabbitMQ(uri string, maxAttempts int) (*amqp.Connection, error) {
	var conn *amqp.Connection
	var err error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		conn, err = amqp.Dial(uri)
		if err == nil {
			return conn, nil
		}

		log.Printf("RabbitMQ not ready yet, attempt %d/%d: %v", attempt, maxAttempts, err)
		time.Sleep(time.Duration(attempt) * 2 * time.Second)
	}

	return nil, err
}

type CustomerInfo struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	TaxID      string `json:"tax_id"` // CPF or CNPJ
	Address    string `json:"address"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
}

type paymentRequestedMessage struct {
	EventName      string        `json:"event_name"`
	PaymentID      string        `json:"payment_id"`
	IdempotencyKey string        `json:"idempotency_key"`
	AmountCents    int64         `json:"amount_cents"`
	Currency       string        `json:"currency"`
	PaymentMethod  string        `json:"payment_method"`
	Customer       *CustomerInfo `json:"customer,omitempty"`
	Installments   *int          `json:"installments,omitempty"`
	OccurredAt     string        `json:"occurred_at"`
}

func StartWorker(ctx context.Context, pool *pgxpool.Pool, gateway paymentgateway.Gateway, cfg *config.Config) {
	conn, err := connectRabbitMQ(cfg.RabbitmqURI, 10)
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	fmt.Println("channel opened successfully: ", ch)

	q, err := ch.QueueDeclare(
		cfg.RabbitmqQueue, // name
		true,              // durability
		false,             // delete when unused
		false,             // exclusive
		false,             // no-wait
		amqp.Table{
			amqp.QueueTypeArg: amqp.QueueTypeQuorum,
		},
	)
	failOnError(err, "Failed to declare a queue")

	fmt.Println("Queue declared successfully: ", q)

	consumerTag := "payment-consumer"
	msgs, err := ch.Consume(
		q.Name, // queue
		consumerTag,
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	failOnError(err, "Failed to register a consumer")
	fmt.Println("message from channel consumer: ", msgs)

	queries := bridge.New(pool)
	processor := NewPaymentRequestedProcessor(queries, gateway, cfg)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)
			err := processor.Handle(context.Background(), d)
			if err != nil {
				log.Printf("failed to handle payment requested message: %v", err)
			}
			ackOrNack(d, err)
		}
	}()

	<-ctx.Done()
	log.Println("shutdown requested, stopping consumer...")
	if err := ch.Cancel(consumerTag, false); err != nil {
		log.Printf("failed to cancel consumer: %v", err)
	}
	wg.Wait()
	log.Println("consumer stopped")
}

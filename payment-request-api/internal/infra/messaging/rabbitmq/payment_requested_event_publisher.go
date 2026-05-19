package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/payment/events"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQPaymentRequestedEventPublisher struct {
	URI        string
	Exchange   string
	Queue      string
	RoutingKey string
}

func NewRabbitMQPaymentRequestedEventPublisher(uri, exchange, queue, routingKey string) *RabbitMQPaymentRequestedEventPublisher {
	return &RabbitMQPaymentRequestedEventPublisher{
		URI:        uri,
		Exchange:   exchange,
		Queue:      queue,
		RoutingKey: routingKey,
	}
}

func (p *RabbitMQPaymentRequestedEventPublisher) Publish(event *events.PaymentRequestedEvent) error {
	if event == nil {
		return fmt.Errorf("nil payment requested event")
	}

	conn, err := amqp.Dial(p.URI)
	if err != nil {
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	exchange := p.Exchange
	if exchange == "" {
		exchange = "payment.events"
	}

	routingKey := p.RoutingKey
	if routingKey == "" {
		routingKey = event.EventName
	}

	if err := ch.ExchangeDeclare(
		exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return err
	}

	queue := p.Queue
	if queue == "" {
		queue = "payment_requests"
	}

	if _, err := ch.QueueDeclare(
		queue,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return err
	}

	if err := ch.QueueBind(
		queue,
		routingKey,
		exchange,
		false,
		nil,
	); err != nil {
		return err
	}

	return ch.PublishWithContext(
		context.Background(),
		exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         payload,
			Timestamp:    event.OccurredAt,
			DeliveryMode: amqp.Persistent,
		},
	)
}

package worker

import "strings"

import amqp "github.com/rabbitmq/amqp091-go"

type deliveryOutcome struct {
	ack     bool
	requeue bool
}

func decideDeliveryOutcome(err error) deliveryOutcome {
	if err == nil {
		return deliveryOutcome{ack: true}
	}

	if isRetryableStripeError(err) {
		return deliveryOutcome{ack: false, requeue: true}
	}

	return deliveryOutcome{ack: true}
}

func ackOrNack(d amqp.Delivery, err error) {
	outcome := decideDeliveryOutcome(err)
	if outcome.ack {
		_ = d.Ack(false)
		return
	}

	_ = d.Nack(false, outcome.requeue)
}

func shouldConfirmPaymentIntent(paymentMethod string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(paymentMethod)), "pm_")
}

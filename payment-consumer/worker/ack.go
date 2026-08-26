// payment-consumer\worker\ack.go
package worker

import amqp "github.com/rabbitmq/amqp091-go"

type deliveryOutcome struct {
	ack     bool
	requeue bool
}

// decideDeliveryOutcome determines whether to acknowledge the message or not based on the error.
// If the error is nil, it returns an outcome to acknowledge the message.
func decideDeliveryOutcome(err error) deliveryOutcome {
	if err == nil {
		return deliveryOutcome{ack: true}
	}

	if isRetryableWorkerError(err) {
		return deliveryOutcome{ack: false, requeue: true}
	}

	return deliveryOutcome{ack: true}
}

// ackOrNack acknowledges or negatively acknowledges the message based on the error.
func ackOrNack(d amqp.Delivery, err error) {
	outcome := decideDeliveryOutcome(err)
	if outcome.ack {
		_ = d.Ack(false)
		return
	}

	_ = d.Nack(false, outcome.requeue)
}

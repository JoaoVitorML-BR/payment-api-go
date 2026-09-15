// payment-consumer/worker/headers.go
package worker

import amqp "github.com/rabbitmq/amqp091-go"

func redeliveryCount(d amqp.Delivery) int {
	raw, ok := d.Headers["x-death"]
	if !ok {
		return 0
	}

	deaths, ok := raw.([]interface{})
	if !ok {
		return 0
	}

	var maxCount int64
	for _, entry := range deaths {
		table, ok := entry.(amqp.Table)
		if !ok {
			continue
		}
		queueName, _ := table["queue"].(string)
		if queueName != "payment_requests.retry" {
			continue
		}
		if count, ok := table["count"].(int64); ok && count > maxCount {
			maxCount = count
		}
	}

	return int(maxCount)
}

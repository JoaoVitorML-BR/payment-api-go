// payment-request-api\internal\payment\events\publisher.go
package events

type PaymentRequestedEventPublisher interface {
	Publish(event *PaymentRequestedEvent) error
}
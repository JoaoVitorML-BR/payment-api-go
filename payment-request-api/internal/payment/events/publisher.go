package events

type PaymentRequestedEventPublisher interface {
	Publish(event *PaymentRequestedEvent) error
}
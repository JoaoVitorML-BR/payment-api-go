package paymentStripe

import (
	"context"

	"github.com/stripe/stripe-go/v85"
)

func CreatePaymentIntent(sc *Client, amount int64, currency string, idempotencyKey string, metadata map[string]string) (*stripe.PaymentIntent, error) {
	params := &stripe.PaymentIntentCreateParams{
		Amount:   stripe.Int64(amount),
		Currency: stripe.String(currency),
		Metadata: metadata,
		AutomaticPaymentMethods: &stripe.PaymentIntentCreateAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(false),
		},
		PaymentMethodTypes: []*string{stripe.String("card")},
	}
	if idempotencyKey != "" {
		params.SetIdempotencyKey(idempotencyKey)
	}

	result, err := sc.stripeClient.V1PaymentIntents.Create(context.Background(), params)
	return result, err
}

func ConfirmPaymentIntent(sc *Client, id string, paymentMethodID string) (*stripe.PaymentIntent, error) {
    params := &stripe.PaymentIntentConfirmParams{
        PaymentMethod: stripe.String(paymentMethodID),
    }

    result, err := sc.stripeClient.V1PaymentIntents.Confirm(context.Background(), id, params)
    return result, err
}

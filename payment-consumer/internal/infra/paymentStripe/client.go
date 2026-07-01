package paymentStripe

import sdkstripe "github.com/stripe/stripe-go/v85"

type Client struct {
	stripeClient *sdkstripe.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		stripeClient: sdkstripe.NewClient(apiKey),
	}
}

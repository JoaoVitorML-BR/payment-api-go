package paymentmercadopago

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/mercadopago/sdk-go/pkg/config"
	"github.com/mercadopago/sdk-go/pkg/order"
	"github.com/mercadopago/sdk-go/pkg/requestoptions"
)

type RefundClient struct {
	accessToken string
	store       *EncryptedFileTokenStore
}

func NewRefundClient(accessToken string, store *EncryptedFileTokenStore) *RefundClient {
	return &RefundClient{accessToken: strings.TrimSpace(accessToken), store: store}
}

func (c *RefundClient) Refund(ctx context.Context, gatewayPaymentID, sellerID string, amountCents int64) (string, error) {
	client, id, err := c.client(ctx, gatewayPaymentID, sellerID)
	if err != nil {
		return "", err
	}
	result, err := client.Get(ctx, id)
	if err != nil {
		return "", fmt.Errorf("get Mercado Pago order for refund: %w", err)
	}
	if result == nil || len(result.Transactions.Payments) == 0 {
		return "", fmt.Errorf("Mercado Pago order has no payment transaction")
	}
	transactionID := result.Transactions.Payments[0].ID
	var request *order.RefundRequest
	if amountCents > 0 {
		request = &order.RefundRequest{Transactions: []order.RefundTransaction{{ID: transactionID, Amount: fmt.Sprintf("%.2f", float64(amountCents)/100)}}}
	}
	refundResult, err := client.Refund(requestoptions.WithIdempotencyKey(ctx, fmt.Sprintf("refund-%s-%d", gatewayPaymentID, amountCents)), id, request)
	if err != nil {
		return "", fmt.Errorf("create Mercado Pago order refund: %w", err)
	}
	if refundResult == nil || len(refundResult.Transactions.Refunds) == 0 {
		return "", fmt.Errorf("Mercado Pago order refund was not confirmed")
	}
	return refundResult.Transactions.Refunds[0].ID, nil
}

func (c *RefundClient) FindRefund(ctx context.Context, gatewayPaymentID, sellerID string, amountCents int64) (string, bool, error) {
	client, id, err := c.client(ctx, gatewayPaymentID, sellerID)
	if err != nil {
		return "", false, err
	}
	result, err := client.Get(ctx, id)
	if err != nil {
		return "", false, fmt.Errorf("get Mercado Pago order refunds: %w", err)
	}
	for _, item := range result.Transactions.Refunds {
		value, parseErr := strconv.ParseFloat(item.Amount, 64)
		if item.ID != "" && strings.EqualFold(item.Status, "processed") && parseErr == nil && int64(value*100+0.5) == amountCents {
			return item.ID, true, nil
		}
	}
	return "", false, nil
}

func (c *RefundClient) client(ctx context.Context, gatewayPaymentID, sellerID string) (order.Client, string, error) {
	id := strings.TrimSpace(gatewayPaymentID)
	if id == "" {
		return nil, "", fmt.Errorf("invalid Mercado Pago order id %q", gatewayPaymentID)
	}
	accessToken := c.accessToken
	if strings.TrimSpace(sellerID) != "" {
		if c.store == nil {
			return nil, "", fmt.Errorf("seller token store is not configured")
		}
		token, err := c.store.Load(sellerID)
		if err != nil {
			return nil, "", err
		}
		accessToken = token.AccessToken
	}
	if accessToken == "" {
		return nil, "", fmt.Errorf("Mercado Pago access token is not configured")
	}
	cfg, err := config.New(accessToken)
	if err != nil {
		return nil, "", err
	}
	return order.NewClient(cfg), id, nil
}

package paymentmercadopago

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/JoaoVitorML-BR/payment-api-go/payment-consumer/internal/infra/paymentgateway"
	"github.com/mercadopago/sdk-go/pkg/config"
	"github.com/mercadopago/sdk-go/pkg/order"
	"github.com/mercadopago/sdk-go/pkg/requestoptions"
)

func (c *Client) CreatePayment(ctx context.Context, input paymentgateway.CreatePaymentInput) (*paymentgateway.PaymentResult, error) {
	if input.PaymentMethod != "pix" {
		return nil, fmt.Errorf("paymentmercadopago: only 'pix' is implemented so far, got %q", input.PaymentMethod)
	}
	if strings.TrimSpace(input.PayerEmail) == "" || strings.TrimSpace(input.PayerName) == "" || strings.TrimSpace(input.PayerTaxID) == "" {
		return nil, fmt.Errorf("paymentmercadopago: payer email, name, and tax id are required for pix payments")
	}

	ctx = requestoptions.WithIdempotencyKey(ctx, input.IdempotencyKey)

	firstName, lastName := splitFullName(input.PayerName)
	cleanTaxID := cleanDigits(input.PayerTaxID)
	identificationType := identificationTypeForTaxID(cleanTaxID)
	if identificationType == "" {
		return nil, fmt.Errorf("paymentmercadopago: payer tax id must be a valid CPF or CNPJ for pix payments")
	}

	streetName, streetNumber := parseStreetAndNumber(input.PayerAddress)
	areaCode, phoneNumber := parsePhone(input.PayerPhone)
	zipCode := cleanDigits(input.PayerPostalCode)

	sdkConfig := c.cfg
	if strings.TrimSpace(input.SellerID) != "" {
		if input.MarketplaceFeeCents <= 0 {
			return nil, fmt.Errorf("paymentmercadopago: marketplace fee is required for seller split")
		}
		sellerToken, err := c.sellerAccessToken(input.SellerID)
		if err != nil {
			return nil, fmt.Errorf("paymentmercadopago: resolve seller token: %w", err)
		}
		sdkConfig, err = config.New(sellerToken)
		if err != nil {
			return nil, fmt.Errorf("paymentmercadopago: create seller config: %w", err)
		}
	} else if input.MarketplaceFeeCents > 0 {
		return nil, fmt.Errorf("paymentmercadopago: seller id is required for marketplace fee")
	}
	client := order.NewClient(sdkConfig)

	amount := float64(input.AmountCents) / 100

	var marketplaceFee string
	if input.MarketplaceFeeCents > 0 {
		marketplaceFee = fmt.Sprintf("%.2f", float64(input.MarketplaceFeeCents)/100)
	}

	itemTitle := strings.TrimSpace(input.Description)
	if itemTitle == "" {
		itemTitle = "Serviço de Consultoria"
	}

	now := time.Now().UTC()
	regDate := now.AddDate(-1, 0, 0).Format("2006-01-02T15:04:05.000-07:00")
	lastPurchaseDate := now.Format("2006-01-02T15:04:05.000-07:00")

	request := order.Request{
		Type:              "online",
		ProcessingMode:    "automatic",
		TotalAmount:       fmt.Sprintf("%.2f", amount),
		Currency:          input.Currency,
		Description:       input.Description,
		ExternalReference: input.Metadata["payment_request_uuid"],
		MarketPlaceFee:    marketplaceFee,
		ExpirationTime:    "PT30M",
		Config: &order.ConfigRequest{
			StatementDescriptor: "CONSULTORIA",
		},
		Items: []order.ItemsRequest{
			{
				Title:        itemTitle,
				UnitPrice:    fmt.Sprintf("%.2f", amount),
				Quantity:     1,
				CategoryID:   "services",
				Description:  itemTitle,
				ExternalCode: "ITEM-001",
			},
		},
		Payer: &order.PayerRequest{
			Email:     strings.TrimSpace(input.PayerEmail),
			FirstName: firstName,
			LastName:  lastName,
			Identification: &order.IdentificationRequest{
				Type:   identificationType,
				Number: cleanTaxID,
			},
			Phone: &order.PhoneRequest{
				AreaCode: areaCode,
				Number:   phoneNumber,
			},
			Address: &order.PayerAddressRequest{
				City:         strings.TrimSpace(input.PayerCity),
				State:        strings.TrimSpace(input.PayerState),
				ZipCode:      zipCode,
				StreetName:   streetName,
				StreetNumber: streetNumber,
			},
		},
		AdditionalInfo: &order.AdditionalInfoRequest{
			PayerAuthenticationType:    "WEB",
			PayerRegistrationDate:      regDate,
			PayerIsFirstPurchaseOnLine: true,
			PayerLastPurchase:          lastPurchaseDate,
		},
		Transactions: &order.TransactionRequest{Payments: []order.PaymentRequest{{
			Amount:        fmt.Sprintf("%.2f", amount),
			PaymentMethod: &order.PaymentMethodRequest{ID: "pix", Type: "bank_transfer"},
		}}},
	}

	result, err := client.Create(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("create pix payment: %w", err)
	}

	return toOrderPaymentResult(result), nil
}

func (c *Client) GetPayment(ctx context.Context, gatewayPaymentID string) (*paymentgateway.PaymentResult, error) {
	client := order.NewClient(c.cfg)
	result, err := client.Get(ctx, strings.TrimSpace(gatewayPaymentID))
	if err != nil {
		return nil, fmt.Errorf("get payment %s: %w", gatewayPaymentID, err)
	}

	return toOrderPaymentResult(result), nil
}

func toOrderPaymentResult(o *order.Response) *paymentgateway.PaymentResult {
	if o == nil {
		return &paymentgateway.PaymentResult{Status: paymentgateway.StatusFailed, RawStatus: "invalid_order"}
	}
	if len(o.Transactions.Payments) == 0 {
		return &paymentgateway.PaymentResult{
			GatewayPaymentID: o.ID,
			Status:           normalizeOrderStatus(o.Status),
			RawStatus:        o.Status,
			Currency:         o.Currency,
			RawResponse:      mustJSON(o),
		}
	}
	p := o.Transactions.Payments[0]
	amount, _ := strconv.ParseFloat(p.Amount, 64)
	expiration := p.DateOfExpiration
	if expiration == "" {
		expiration = p.ExpirationTime
	}
	if expiration == "" {
		expiration = o.ExpirationTime
	}
	status := p.Status
	if strings.TrimSpace(status) == "" {
		status = o.Status
	}
	return &paymentgateway.PaymentResult{
		GatewayPaymentID:  o.ID,
		Status:            normalizeOrderStatus(status),
		RawStatus:         status,
		AmountCents:       int64(amount * 100),
		Currency:          o.Currency,
		PixQRCode:         p.PaymentMethod.QrCode,
		PixQRCodeBase64:   p.PaymentMethod.QrCodeBase64,
		PixExpirationDate: expiration,
		RawResponse:       mustJSON(o),
	}
}

func mustJSON(value any) []byte {
	encoded, _ := json.Marshal(value)
	return encoded
}

func normalizeOrderStatus(status string) paymentgateway.PaymentStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "processed", "accredited", "approved", "paid":
		return paymentgateway.StatusApproved
	case "pending", "in_process", "action_required", "created":
		return paymentgateway.StatusPending
	case "cancelled", "canceled", "rejected", "expired":
		return paymentgateway.StatusRejected
	default:
		return paymentgateway.StatusFailed
	}
}

func splitFullName(fullName string) (string, string) {
	parts := strings.Fields(strings.TrimSpace(fullName))
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], parts[0]
	}
	return parts[0], strings.Join(parts[1:], " ")
}

func cleanDigits(str string) string {
	var sb strings.Builder
	for _, r := range str {
		if r >= '0' && r <= '9' {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func parsePhone(rawPhone string) (string, string) {
	digits := cleanDigits(rawPhone)
	if strings.HasPrefix(digits, "55") && (len(digits) == 12 || len(digits) == 13) {
		digits = digits[2:]
	}
	if len(digits) >= 10 {
		return digits[:2], digits[2:]
	}
	if len(digits) > 2 {
		return digits[:2], digits[2:]
	}
	return "11", digits
}

func parseStreetAndNumber(rawAddress string) (string, string) {
	rawAddress = strings.TrimSpace(rawAddress)
	if rawAddress == "" {
		return "Rua", "1"
	}

	if parts := strings.Split(rawAddress, ","); len(parts) >= 2 {
		street := strings.TrimSpace(parts[0])
		rest := strings.TrimSpace(parts[1])
		numParts := strings.Fields(rest)
		if len(numParts) > 0 {
			numDigits := cleanDigits(numParts[0])
			if numDigits != "" {
				return street, numDigits
			}
			return street, numParts[0]
		}
		return street, "1"
	}

	fields := strings.Fields(rawAddress)
	if len(fields) > 1 {
		lastField := fields[len(fields)-1]
		numDigits := cleanDigits(lastField)
		if numDigits != "" && len(numDigits) <= 6 {
			street := strings.Join(fields[:len(fields)-1], " ")
			return street, numDigits
		}
	}

	return rawAddress, "1"
}

func identificationTypeForTaxID(taxID string) string {
	digits := cleanDigits(taxID)
	switch len(digits) {
	case 11:
		return "CPF"
	case 14:
		return "CNPJ"
	default:
		return ""
	}
}

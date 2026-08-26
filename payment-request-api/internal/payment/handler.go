// payment-request-api\internal\payment\handler.go
package payment

import (
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"encoding/json"

	"github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/infra/webhook"
	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	service *PaymentService // package paymet > service.go > PaymentService
}

type CustomerInfo struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	TaxID      string `json:"tax_id"` // CPF or CNPJ
	Address    string `json:"address"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
}

type CreatePaymentRequest struct {
	IdempotencyKey        string        `json:"idempotency_key" binding:"required"`
	MerchantReference     string        `json:"merchant_reference"`
	AmountCents           int64         `json:"amount_cents" binding:"required,gt=0"`
	Currency              string        `json:"currency" binding:"required,len=3"`
	PaymentMethod         string        `json:"payment_method" binding:"required"`
	StripePaymentMethodID string        `json:"stripe_payment_method_id,omitempty"`
	Installments          *int          `json:"installments,omitempty"`
	Customer              *CustomerInfo `json:"customer,omitempty"`
}

// router use this func to create a new instance of PaymentHandler and inject the PaymentService dependency,
// this way we can keep the handler decoupled from the service and make it easier to test and maintain in the future.
func NewPaymentHandler(service *PaymentService) (*PaymentHandler, error) {
	if service == nil {
		return nil, errors.New("nil service provided to NewPaymentHandler")
	}
	return &PaymentHandler{service: service}, nil
}

func (h *PaymentHandler) RefundHandler(c *gin.Context) {
	var req RefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.ProcessRefund(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "refund processed"})
}

func (h *PaymentHandler) GetPaymentClientSecretHandler(c *gin.Context) {
	paymentUUID := c.Param("payment_id")
	if paymentUUID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment_id must be provided"})
		return
	}

	paymentStatus, err := h.service.GetPaymentClientSecret(c.Request.Context(), paymentUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment attempt not found"})
		return
	}
	c.JSON(http.StatusOK, paymentStatus)
}

func (h *PaymentHandler) CreatePaymentRequestHandler(c *gin.Context) {
	var req CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	paymentRequest, err := h.service.CreatePayment(c.Request.Context(), CreatePaymentRequest{
		IdempotencyKey:        req.IdempotencyKey,
		MerchantReference:     req.MerchantReference,
		AmountCents:           req.AmountCents,
		Currency:              req.Currency,
		PaymentMethod:         req.PaymentMethod,
		StripePaymentMethodID: req.StripePaymentMethodID,
		Installments:          req.Installments,
		Customer:              req.Customer,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "payment request accepted",
		"data":    paymentRequest,
	})
}

// MercadoPagoWebhookHandler receives Mercado Pago notifications.
//
// SECURITY RULES (see Readme.ctx.md):
//   1. The webhook payload is NEVER trusted for financial state. Only the
//      "data.id" is extracted and used to re-query Mercado Pago.
//   2. The X-Signature header is verified using the official Mercado Pago
//      manifest (id:<data.id>;request-id:<x-request-id>;ts:<ts>;).
//   3. The service layer validates external_reference, amount and currency
//      against the local database before any status change.
//
// Response contract with Mercado Pago:
//   - 200/201: notification processed (or intentionally ignored) — stop retries.
//   - 4xx (except 401/403): permanent rejection — stop retries.
//   - 5xx / 408: temporary failure — Mercado Pago will retry with backoff.
func (h *PaymentHandler) MercadoPagoWebhookHandler(c *gin.Context) {
	xSignature := c.GetHeader("X-Signature")
	xRequestID := c.GetHeader("X-Request-Id")
	if xSignature == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing X-Signature header"})
		return
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(c.Request.Body, maxWebhookBodySize))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to read request body"})
		return
	}

	dataID := extractWebhookDataID(bodyBytes)
	if dataID == "" {
		// Malformed/unrecognized payloads are permanently rejected so that
		// Mercado Pago does not keep retrying a message we can never process.
		log.Printf("[WEBHOOK] rejected: missing data.id in payload")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook payload"})
		return
	}

	if err := webhook.VerifySignature(xSignature, xRequestID, dataID, time.Now()); err != nil {
		log.Printf("[WEBHOOK] signature verification failed: %v", err)
		c.JSON(http.StatusForbidden, gin.H{"error": "Invalid signature"})
		return
	}

	// Never trust the webhook status — re-query Mercado Pago using only the id.
	if err := h.service.ProcessMercadoPagoWebhook(c.Request.Context(), dataID); err != nil {
		switch {
		case errors.Is(err, ErrWebhookRetryable):
			// Transient failure (local row not linked yet, gateway timeout...).
			// Return 5xx so Mercado Pago retries later.
			log.Printf("[WEBHOOK] retryable error for payment %s: %v", dataID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "temporary failure, retry later"})
		case errors.Is(err, ErrWebhookInvalidState):
			// Permanent rejection (external_reference/amount/currency mismatch or
			// invalid state transition). Do not retry.
			log.Printf("[WEBHOOK] invalid state for payment %s: %v", dataID, err)
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "webhook event rejected"})
		default:
			log.Printf("[WEBHOOK] unexpected error for payment %s: %v", dataID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

const maxWebhookBodySize = 64 * 1024 // 64 KiB is far more than enough for MP notifications

// extractWebhookDataID extracts the resource id from a Mercado Pago webhook
// body without trusting any other field of the payload.
func extractWebhookDataID(body []byte) string {
	var event struct {
		Data struct {
			ID json.Number `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &event); err != nil {
		return ""
	}
	return event.Data.ID.String()
}
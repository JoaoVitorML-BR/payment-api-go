// payment-request-api\internal\payment\handler.go
package payment

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/config"
	"github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/infra/webhook"
	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	service *PaymentService // package paymet > service.go > PaymentService
	config  *config.Config
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
	SellerID              string        `json:"seller_id,omitempty"`
	MarketplaceFeeCents   int64         `json:"marketplace_fee_cents,omitempty"`
	Customer              *CustomerInfo `json:"customer,omitempty"`
}

// router use this func to create a new instance of PaymentHandler and inject the PaymentService dependency,
// this way we can keep the handler decoupled from the service and make it easier to test and maintain in the future.
func NewPaymentHandler(service *PaymentService, cfg *config.Config) (*PaymentHandler, error) {
	if service == nil {
		return nil, errors.New("nil service provided to NewPaymentHandler")
	}
	if cfg == nil {
		return nil, errors.New("nil config provided to NewPaymentHandler")
	}
	return &PaymentHandler{service: service, config: cfg}, nil
}

func (h *PaymentHandler) RefundHandler(c *gin.Context) {
	var req RefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.IdempotencyKey == "" {
		req.IdempotencyKey = c.GetHeader("Idempotency-Key")
		if req.IdempotencyKey == "" {
			req.IdempotencyKey = c.GetHeader("X-Idempotency-Key")
		}
	}

	if err := h.service.ProcessRefund(c.Request.Context(), req); err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "cannot be refunded") || strings.Contains(err.Error(), "exceeds") {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
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

func (h *PaymentHandler) GetPaymentStatusHandler(c *gin.Context) {
	paymentID := strings.TrimSpace(c.Param("payment_id"))
	if paymentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment_id must be provided"})
		return
	}

	statusComparison, err := h.service.GetPaymentStatusComparison(c.Request.Context(), paymentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, statusComparison)
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
		SellerID:              req.SellerID,
		MarketplaceFeeCents:   req.MarketplaceFeeCents,
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
//  1. The webhook payload is NEVER trusted for financial state. Only the
//     "data.id" is extracted and used to re-query Mercado Pago.
//  2. The X-Signature header is verified using the official Mercado Pago
//     manifest (id:<data.id>;request-id:<x-request-id>;ts:<ts>;).
//  3. The service layer validates external_reference, amount and currency
//     against the local database before any status change.
//
// Response contract with Mercado Pago:
//   - 200/201: notification processed (or intentionally ignored) — stop retries.
//   - 4xx (except 401/403): permanent rejection — stop retries.
//   - 5xx / 408: temporary failure — Mercado Pago will retry with backoff.
func (h *PaymentHandler) MercadoPagoWebhookHandler(c *gin.Context) {
	for key, values := range c.Request.Header {
		log.Printf("[WEBHOOK] HEADER %s: %v", key, values)
	}

	fmt.Printf("***WebehookSecretLocal***: %s\n", h.config.MercadoPagoWebhookSecret)

	xSignature := c.GetHeader("X-Signature")
	xRequestID := c.GetHeader("X-Request-Id")

	if xSignature == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing X-Signature header",
		})
		return
	}

	bodyBytes, err := io.ReadAll(
		io.LimitReader(c.Request.Body, maxWebhookBodySize),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Unable to read request body",
		})
		return
	}

	log.Printf("[WEBHOOK] Raw URL Query: %s", c.Request.URL.RawQuery)
	log.Printf("[WEBHOOK] Raw Body: %s", string(bodyBytes))

	dataID := strings.TrimSpace(c.Query("data.id"))

	if dataID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing data.id",
		})
		return
	}

	skipSignature := os.Getenv("WEBHOOK_SKIP_SIGNATURE") == "true" &&
		!isLiveModeWebhook(bodyBytes)

	if skipSignature {

		log.Printf(
			"[WEBHOOK] WARNING: signature check skipped (test mode, WEBHOOK_SKIP_SIGNATURE=true)",
		)

	} else {

		if err := webhook.VerifySignature(
			xSignature,
			xRequestID,
			dataID,
			time.Now(),
		); err != nil {

			log.Printf(
				"[WEBHOOK] signature verification failed: dataID=%s requestID=%s error=%v",
				dataID,
				xRequestID,
				err,
			)

			c.JSON(http.StatusForbidden, gin.H{
				"error": "Invalid signature",
			})
			return
		}

		log.Printf(
			"[WEBHOOK] signature verified successfully: dataID=%s requestID=%s",
			dataID,
			xRequestID,
		)
	}

	if dataID == "123456" {

		log.Printf(
			"[WEBHOOK] simulation test ping received and signature successfully verified!",
		)

		c.JSON(http.StatusOK, gin.H{
			"status": "simulation_verified",
		})
		return
	}

	if err := h.service.ProcessMercadoPagoWebhook(
		c.Request.Context(),
		dataID,
	); err != nil {

		switch {

		case errors.Is(err, ErrWebhookRetryable):
			log.Printf(
				"[WEBHOOK] retryable error for payment %s: %v",
				dataID,
				err,
			)

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "temporary failure, retry later",
			})

		case errors.Is(err, ErrWebhookInvalidState):

			log.Printf(
				"[WEBHOOK] invalid state for payment %s: %v",
				dataID,
				err,
			)

			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error": "webhook event rejected",
			})

		default:

			log.Printf(
				"[WEBHOOK] unexpected error for payment %s: %v",
				dataID,
				err,
			)

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal error",
			})
		}

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
	})
}

const maxWebhookBodySize = 64 * 1024 // 64 KiB is far more than enough for MP notifications

// extractWebhookDataID extracts the primary resource id from a Mercado Pago webhook
// body without trusting any other field of the payload. It supports both string and numeric IDs,
// nested in data.id or top-level id.
func extractWebhookDataID(body []byte) string {
	ids := extractWebhookCandidateIDs(body)
	if len(ids) > 0 {
		return ids[0]
	}
	return ""
}

// extractWebhookCandidateIDs extracts all potential resource IDs (e.g. order ID, payment IDs, top-level ID)
// from the webhook payload so signature verification can test candidate IDs.
func extractWebhookCandidateIDs(body []byte) []string {
	if len(body) == 0 {
		return nil
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil
	}

	seen := make(map[string]bool)
	var ids []string
	addID := func(v any) {
		if v == nil {
			return
		}
		var str string
		switch val := v.(type) {
		case string:
			str = strings.TrimSpace(val)
		case float64:
			str = strconv.FormatInt(int64(val), 10)
		case json.Number:
			str = val.String()
		default:
			str = strings.TrimSpace(fmt.Sprintf("%v", val))
		}
		if str != "" && !seen[str] {
			seen[str] = true
			ids = append(ids, str)
		}
	}

	// 1. Check data.id
	if data, ok := raw["data"].(map[string]any); ok {
		if id, exists := data["id"]; exists {
			addID(id)
		}
		// Check nested transaction payments IDs
		if txs, ok := data["transactions"].(map[string]any); ok {
			if payments, ok := txs["payments"].([]any); ok {
				for _, p := range payments {
					if pMap, ok := p.(map[string]any); ok {
						if pid, exists := pMap["id"]; exists {
							addID(pid)
						}
					}
				}
			}
		}
	}

	// 2. Check top-level id
	if id, exists := raw["id"]; exists {
		addID(id)
	}

	return ids
}

func isLiveModeWebhook(body []byte) bool {
	var p struct {
		LiveMode bool `json:"live_mode"`
	}
	if err := json.Unmarshal(body, &p); err != nil {
		return true
	}
	return p.LiveMode
}

// Controller
package payment

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	service *PaymentService // package paymet > service.go > PaymentService
}

type CreatePaymentRequest struct {
	IdempotencyKey    string `json:"idempotency_key" binding:"required"`
	MerchantReference string `json:"merchant_reference"`
	AmountCents       int64  `json:"amount_cents" binding:"required,gt=0"`
	Currency          string `json:"currency" binding:"required,len=3"`
	PaymentMethod     string `json:"payment_method" binding:"required"`
	Installments      *int   `json:"installments,omitempty"`
}

// router use this func to create a new instance of PaymentHandler and inject the PaymentService dependency, 
// this way we can keep the handler decoupled from the service and make it easier to test and maintain in the future.
func NewPaymentHandler(service *PaymentService) (*PaymentHandler, error) {
	if service == nil {
		return nil, errors.New("nil service provided to NewPaymentHandler")
	}
	return &PaymentHandler{service: service}, nil
}

func (h *PaymentHandler) CreatePaymentRequestHandler(c *gin.Context) {
	var req CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	paymentRequest, err := h.service.CreatePayment(c.Request.Context(), CreatePaymentRequest{
		IdempotencyKey:    req.IdempotencyKey,
		MerchantReference: req.MerchantReference,
		AmountCents:       req.AmountCents,
		Currency:          req.Currency,
		PaymentMethod:     req.PaymentMethod,
		Installments:      req.Installments,
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
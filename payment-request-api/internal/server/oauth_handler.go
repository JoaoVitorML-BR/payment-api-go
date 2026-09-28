package server

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/JoaoVitorML-BR/payment-api-go/payment-request-api/internal/infra/paymentmercadopago"
	"github.com/gin-gonic/gin"
)

type OAuthHandler struct {
	client       *paymentmercadopago.OAuthClient
	store        paymentmercadopago.SellerTokenStore
	redirectURI  string
	stateMu      sync.Mutex
	pendingState map[string]time.Time
}

func NewOAuthHandler(client *paymentmercadopago.OAuthClient, store paymentmercadopago.SellerTokenStore, redirectURI string) (*OAuthHandler, error) {
	if client == nil || store == nil || strings.TrimSpace(redirectURI) == "" {
		return nil, errors.New("OAuth client, token store, and redirect URI are required")
	}
	return &OAuthHandler{client: client, store: store, redirectURI: redirectURI, pendingState: make(map[string]time.Time)}, nil
}

func (h *OAuthHandler) Start(c *gin.Context) {
	state, err := randomState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create OAuth state"})
		return
	}
	h.stateMu.Lock()
	h.pendingState[state] = time.Now().Add(10 * time.Minute)
	h.stateMu.Unlock()
	location, err := h.client.AuthorizationURL(h.redirectURI, state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create OAuth authorization URL"})
		return
	}
	c.Redirect(http.StatusFound, location)
}

func (h *OAuthHandler) Callback(c *gin.Context) {
	state := strings.TrimSpace(c.Query("state"))
	code := strings.TrimSpace(c.Query("code"))
	if !h.consumeState(state) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired OAuth state"})
		return
	}
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing OAuth authorization code"})
		return
	}
	token, err := h.client.ExchangeCode(c.Request.Context(), code, h.redirectURI)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Mercado Pago OAuth exchange failed"})
		return
	}
	if err := h.store.Save(token.UserID, token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not store seller credentials"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "seller connected", "seller_id": token.UserID})
}

func (h *OAuthHandler) consumeState(state string) bool {
	h.stateMu.Lock()
	defer h.stateMu.Unlock()
	expiresAt, ok := h.pendingState[state]
	if ok {
		delete(h.pendingState, state)
	}
	return ok && time.Now().Before(expiresAt)
}

func randomState() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

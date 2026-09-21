package paymentmercadopago

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestOAuthClientAuthorizationURL(t *testing.T) {
	client, err := newOAuthClient("app-123", "secret", "https://api.example", http.DefaultClient)
	if err != nil {
		t.Fatal(err)
	}

	got, err := client.AuthorizationURL("https://example.test/callback", "state-123")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Query().Get("client_id") != "app-123" || parsed.Query().Get("state") != "state-123" {
		t.Fatalf("unexpected authorization URL: %s", got)
	}
}

func TestOAuthClientExchangeCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/oauth/token" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := r.ParseForm(); err != nil || r.Form.Get("code") != "code-123" {
			t.Fatalf("unexpected form: %v", r.Form)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"access","refresh_token":"refresh","user_id":"seller-1"}`))
	}))
	defer server.Close()

	client, err := newOAuthClient("app", "secret", server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	token, err := client.ExchangeCode(context.Background(), "code-123", "https://example.test/callback")
	if err != nil {
		t.Fatal(err)
	}
	if token.AccessToken != "access" || token.RefreshToken != "refresh" || token.UserID != "seller-1" {
		t.Fatalf("unexpected token: %+v", token)
	}
	if strings.Contains(token.AccessToken, "secret") {
		t.Fatal("token unexpectedly contains client secret")
	}
}

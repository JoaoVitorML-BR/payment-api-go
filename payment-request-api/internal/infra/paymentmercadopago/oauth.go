package paymentmercadopago

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const defaultOAuthBaseURL = "https://api.mercadopago.com"

type OAuthToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	UserID       string `json:"user_id"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

func (t *OAuthToken) UnmarshalJSON(data []byte) error {
	type Alias OAuthToken
	var aux struct {
		Alias
		UserID interface{} `json:"user_id"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*t = OAuthToken(aux.Alias)
	switch v := aux.UserID.(type) {
	case string:
		t.UserID = strings.TrimSpace(v)
	case float64:
		t.UserID = strconv.FormatInt(int64(v), 10)
	case json.Number:
		t.UserID = v.String()
	case nil:
		t.UserID = ""
	default:
		t.UserID = fmt.Sprintf("%v", v)
	}
	return nil
}

type OAuthClient struct {
	clientID     string
	clientSecret string
	baseURL      string
	httpClient   *http.Client
}

func NewOAuthClient(clientID, clientSecret string) (*OAuthClient, error) {
	return newOAuthClient(clientID, clientSecret, defaultOAuthBaseURL, http.DefaultClient)
}

func newOAuthClient(clientID, clientSecret, baseURL string, httpClient *http.Client) (*OAuthClient, error) {
	if strings.TrimSpace(clientID) == "" || strings.TrimSpace(clientSecret) == "" {
		return nil, errors.New("Mercado Pago OAuth client credentials are required")
	}
	if strings.TrimSpace(baseURL) == "" {
		return nil, errors.New("Mercado Pago OAuth base URL is required")
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &OAuthClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		baseURL:      strings.TrimRight(baseURL, "/"),
		httpClient:   httpClient,
	}, nil
}

func (c *OAuthClient) AuthorizationURL(redirectURI, state string) (string, error) {
	if strings.TrimSpace(redirectURI) == "" || strings.TrimSpace(state) == "" {
		return "", errors.New("redirect URI and state are required")
	}

	query := url.Values{
		"client_id":     {c.clientID},
		"response_type": {"code"},
		"platform_id":   {"mp"},
		"redirect_uri":  {redirectURI},
		"state":         {state},
	}
	return "https://auth.mercadopago.com.br/authorization?" + query.Encode(), nil
}

func (c *OAuthClient) ExchangeCode(ctx context.Context, code, redirectURI string) (OAuthToken, error) {
	if strings.TrimSpace(code) == "" || strings.TrimSpace(redirectURI) == "" {
		return OAuthToken{}, errors.New("authorization code and redirect URI are required")
	}

	form := url.Values{
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return OAuthToken{}, fmt.Errorf("create OAuth token request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+c.clientSecret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return OAuthToken{}, fmt.Errorf("exchange Mercado Pago OAuth code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return OAuthToken{}, fmt.Errorf("Mercado Pago OAuth token exchange returned HTTP %d", resp.StatusCode)
	}

	var token OAuthToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return OAuthToken{}, fmt.Errorf("decode Mercado Pago OAuth token response: %w", err)
	}
	if strings.TrimSpace(token.AccessToken) == "" || strings.TrimSpace(token.RefreshToken) == "" || strings.TrimSpace(token.UserID) == "" {
		return OAuthToken{}, errors.New("Mercado Pago OAuth response did not contain required credentials")
	}
	return token, nil
}

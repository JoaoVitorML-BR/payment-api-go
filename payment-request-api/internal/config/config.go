// payment-request-api\internal\config\config.go
package config

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type Config struct {
	Port                          string
	Pool                          *pgxpool.Pool
	MercadoPagoWebhookSecret      string
	MercadoPagoOAuthClientID      string
	MercadoPagoOAuthClientSecret  string
	MercadoPagoOAuthRedirectURI   string
	MercadoPagoOAuthTokenFile     string
	MercadoPagoOAuthEncryptionKey string
}

func LoadConfig() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error on load .env:", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Println("PORT not set, using default:", port)
	}

	dbHost := os.Getenv("DB_PS_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}

	dbPort := os.Getenv("DB_PS_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}

	dbUser := os.Getenv("DB_PS_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}

	dbPassword := os.Getenv("DB_PS_PASSWORD")
	if dbPassword == "" {
		dbPassword = "postgres"
	}

	dbName := os.Getenv("DB_PS_DATABASE")
	if dbName == "" {
		dbName = "payment_request"
	}

	mercadoPagoWebhookSecret := os.Getenv("MERCADO_PAGO_WEBHOOK_SECRET")
	if mercadoPagoWebhookSecret == "" {
		return nil, fmt.Errorf(
			"MERCADO_PAGO_WEBHOOK_SECRET is not set in the environment",
		)
	}

	oauthClientID := os.Getenv("MERCADO_PAGO_OAUTH_CLIENT_ID")
	oauthClientSecret := os.Getenv("MERCADO_PAGO_OAUTH_CLIENT_SECRET")
	oauthRedirectURI := os.Getenv("MERCADO_PAGO_OAUTH_REDIRECT_URI")
	oauthTokenFile := os.Getenv("MERCADO_PAGO_OAUTH_TOKEN_FILE")
	oauthEncryptionKey := os.Getenv("MERCADO_PAGO_OAUTH_ENCRYPTION_KEY")
	oauthValues := []string{oauthClientID, oauthClientSecret, oauthRedirectURI, oauthTokenFile, oauthEncryptionKey}
	oauthConfigured := false
	for _, value := range oauthValues {
		if value != "" {
			oauthConfigured = true
			break
		}
	}
	if oauthConfigured {
		for name, value := range map[string]string{
			"MERCADO_PAGO_OAUTH_CLIENT_ID":      oauthClientID,
			"MERCADO_PAGO_OAUTH_CLIENT_SECRET":  oauthClientSecret,
			"MERCADO_PAGO_OAUTH_REDIRECT_URI":   oauthRedirectURI,
			"MERCADO_PAGO_OAUTH_TOKEN_FILE":     oauthTokenFile,
			"MERCADO_PAGO_OAUTH_ENCRYPTION_KEY": oauthEncryptionKey,
		} {
			if value == "" {
				return nil, fmt.Errorf("%s is required when Mercado Pago OAuth is enabled", name)
			}
		}
	}

	// Create pgxpool connection
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", dbUser, dbPassword, dbHost, dbPort, dbName)
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, err
	}

	// Test connection
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, err
	}

	cfg := &Config{
		Port:                          port,
		Pool:                          pool,
		MercadoPagoWebhookSecret:      mercadoPagoWebhookSecret,
		MercadoPagoOAuthClientID:      oauthClientID,
		MercadoPagoOAuthClientSecret:  oauthClientSecret,
		MercadoPagoOAuthRedirectURI:   oauthRedirectURI,
		MercadoPagoOAuthTokenFile:     oauthTokenFile,
		MercadoPagoOAuthEncryptionKey: oauthEncryptionKey,
	}
	return cfg, nil
}

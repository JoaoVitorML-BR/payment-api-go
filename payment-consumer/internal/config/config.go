// payment-consumer\internal\config\config.go
package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                   string
	DbHost                 string
	DbPort                 string
	DbUser                 string
	DbPassword             string
	DbName                 string
	RabbitmqURI            string
	RabbitmqQueue          string
	MercadoPagoAccessToken string
	MercadoPagoWebhookURL  string
}

func LoadConfig() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error on load .env:", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
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

	rabbitmqURI := os.Getenv("RABBITMQ_URL")
	if rabbitmqURI == "" {
		rabbitmqURI = "amqp://guest:guest@localhost:5672/"
	}

	rabbitmqQueue := os.Getenv("RABBITMQ_QUEUE")
	if rabbitmqQueue == "" {
		rabbitmqQueue = "payment_requests"
	}

	mercadoPagoAccessToken := os.Getenv("MERCADO_PAGO_ACCESS_TOKEN")
	if strings.TrimSpace(mercadoPagoAccessToken) == "" {
		return nil, fmt.Errorf("MERCADO_PAGO_ACCESS_TOKEN is required")
	}

	mercadoPagoWebhookURL := os.Getenv("MERCADO_PAGO_WEBHOOK_URL")
	if strings.TrimSpace(mercadoPagoWebhookURL) == "" {
		return nil, fmt.Errorf("MERCADO_PAGO_WEBHOOK_URL is required")
	}

	cfg := &Config{
		Port:                   port,
		DbHost:                 dbHost,
		DbPort:                 dbPort,
		DbUser:                 dbUser,
		DbPassword:             dbPassword,
		DbName:                 dbName,
		RabbitmqURI:            rabbitmqURI,
		RabbitmqQueue:          rabbitmqQueue,
		MercadoPagoAccessToken: mercadoPagoAccessToken,
		MercadoPagoWebhookURL:  mercadoPagoWebhookURL,
	}
	return cfg, nil
}

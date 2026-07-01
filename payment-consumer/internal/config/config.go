package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port          string
	DbHost        string
	DbPort        string
	DbUser        string
	DbPassword    string
	DbName        string
	RabbitmqURI   string
	RabbitmqQueue string
	StripeAPIKey  string
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

	stripeAPIKey := os.Getenv("STRIPE_API_KEY")
	if strings.TrimSpace(stripeAPIKey) == "" {
		return nil, fmt.Errorf("STRIPE_API_KEY is required")
	}

	cfg := &Config{
		Port:          port,
		DbHost:        dbHost,
		DbPort:        dbPort,
		DbUser:        dbUser,
		DbPassword:    dbPassword,
		DbName:        dbName,
		RabbitmqURI:   rabbitmqURI,
		RabbitmqQueue: rabbitmqQueue,
		StripeAPIKey:  stripeAPIKey,
	}
	return cfg, nil
}

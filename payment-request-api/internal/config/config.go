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
	Port string
	Pool *pgxpool.Pool
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
		Port: port,
		Pool: pool,
	}
	return cfg, nil
}

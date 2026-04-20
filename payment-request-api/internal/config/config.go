package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port string
}

func LoadConfig() (*Config, error) {

	err := godotenv.Load()
	if err != nil {
		log.Println("Error on load .env:", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port if not set
		log.Println("PORT not set, using default:", port)
	}

	cfg := &Config{Port: port}
	return cfg, nil
}

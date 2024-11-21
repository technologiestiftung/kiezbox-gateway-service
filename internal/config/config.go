package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func LoadConfig() (string, string, string, string) {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Fetch InfluxDB configuration from environment variables
	url := os.Getenv("INFLUXDB_URL")
	token := os.Getenv("INFLUXDB_TOKEN")
	org := os.Getenv("INFLUXDB_ORG")
	bucket := os.Getenv("INFLUXDB_BUCKET")

	// Check if any required variable is missing
	if url == "" || token == "" || org == "" || bucket == "" {
		log.Fatal("Missing one or more required environment variables")
	}

	return url, token, org, bucket
}

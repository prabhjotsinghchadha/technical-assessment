package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func LoadDotEnv() {
	if os.Getenv("DATABASE_URL") != "" && os.Getenv("APP_HOST") != "" {
		return
	}
	paths := []string{".env", "../.env", "../../.env", "../../../.env"}
	for _, p := range paths {
		if err := godotenv.Load(p); err == nil {
			return
		}
	}
	log.Fatal("Error while loading .env file")
}

func GetEnv(key string) string {
	LoadDotEnv()
	value, exists := os.LookupEnv(key)
	if !exists {
		log.Fatalf("Environment variable %s not found", key)
	}
	return value
}

package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/ortizdavid/go-bank-core-api/common/config"
	"github.com/ortizdavid/go-bank-core-api/core/controllers"
)

func main() {
	mux := http.NewServeMux()

	dbConn, err := config.NewDBConnectionFromEnv("DATABASE_URL")
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	controllers.RegisterRoutes(mux, dbConn.DB)

	listenAddr := config.ListenAddr()
	fmt.Printf("Listen to: http://%s\n", listenAddr)
	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/BaxtiyorUrolov/Tolov-tizimlari/payme"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: Could not load .env file")
	}

	// Database connection
	db, err := sqlx.ConnectContext(context.Background(), "postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}

	// Payme handler with user-provided key and base URL
	handler := payme.NewHandler(db, os.Getenv("PAYME_KEY"), os.Getenv("PAYME_BASE_URL"))

	// Set up webhook endpoint
	http.HandleFunc("/payme", handler.HandlePaymeWebhook)

	// Example transaction creation
	userID := 123
	amount := 100
	paymeID := os.Getenv("PAYME_MERCHANT_ID")
	returnURL := "http://example.com/callback"

	link, err := handler.CreatePaymeTransaction(userID, amount, paymeID, returnURL)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Println("Payme transaction link:", link)

	// Start server
	log.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

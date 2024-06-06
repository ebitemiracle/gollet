package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"github.com/gin-gonic/gin"
	"go_code/database"
)

// PaystackEvent represents the structure of the webhook event from Paystack
type PaystackEvent struct {
	Event string `json:"event"`
	Data  struct {
		Domain          string  `json:"domain"`
		Status          string  `json:"status"`
		Reference       string  `json:"reference"`
		Amount          float64 `json:"amount"`
		PaidAt          string  `json:"paid_at"`
		Customer        struct {
			CustomerCode string `json:"customer_code"`
		} `json:"customer"`
		Authorization struct {
			Bank        string `json:"bank"`
			AccountName string `json:"account_name"`
		} `json:"authorization"`
	} `json:"data"`
}

// Response represents the generic response structure
type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result,omitempty"`
}

// WebhookHandler handles the incoming Paystack webhook events
func WebhookHandler(c *gin.Context) {
	secretKey := os.Getenv("PAYSTACK_SECRET_KEY")

	// Only process POST requests with the correct Paystack signature
	if c.Request.Method != http.MethodPost || c.GetHeader("X-Paystack-Signature") == "" {
		log.Println("Invalid request method or missing signature header")
		c.Status(http.StatusBadRequest)
		return
	}

	// Read the request body
	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v\n", err)
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to read request body: " + err.Error(),
		})
		return
	}

	// Validate the signature
	signature := c.GetHeader("X-Paystack-Signature")
	if !validateSignature(body, signature, secretKey) {
		log.Println("Invalid signature")
		c.Status(http.StatusUnauthorized)
		return
	}

	// Parse the event
	var event PaystackEvent
	if err := json.Unmarshal(body, &event); err != nil {
		log.Printf("Failed to parse request body: %v\n", err)
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to parse request body: " + err.Error(),
		})
		return
	}

	// Handle the event
	if event.Event == "charge.success" && event.Data.Status == "success" {
		if err := insertTransaction(event); err != nil {
			log.Printf("Failed to insert transaction: %v\n", err)
			c.JSON(http.StatusInternalServerError, Response{
				Status:  "error",
				Message: "Failed to insert transaction: " + err.Error(),
			})
			return
		}
	}

	c.Status(http.StatusOK)
}

// validateSignature checks if the request signature matches the expected signature
func validateSignature(body []byte, signature, secret string) bool {
	hash := hmac.New(sha512.New, []byte(secret))
	hash.Write(body)
	expectedSignature := hex.EncodeToString(hash.Sum(nil))
	return signature == expectedSignature
}

// insertTransaction inserts a successful transaction into the database
func insertTransaction(event PaystackEvent) error {
	db, err := database.PostgreSQLConnect()
	if err != nil {
		log.Printf("Failed to connect to database: %v\n", err)
		return err
	}
	defer db.Close(context.Background())

	tx, err := db.Begin(context.Background())
	if err != nil {
		log.Printf("Failed to begin transaction: %v\n", err)
		return err
	}

	// Insert the new transaction
	_, err = tx.Exec(context.Background(),
		"INSERT INTO user_transaction (reference, amount, created_at, bank, account_name, customer_code, transaction_type) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		event.Data.Reference, event.Data.Amount/100, event.Data.PaidAt, event.Data.Authorization.Bank, event.Data.Authorization.AccountName, event.Data.Customer.CustomerCode, "credit")

	if err != nil {
		tx.Rollback(context.Background())
		log.Printf("Failed to insert transaction: %v\n", err)
		return fmt.Errorf("failed to insert transaction: %w", err)
	}

	// Update previous_balance and current_balance
	_, err = tx.Exec(context.Background(),
		"UPDATE wallet SET previous_balance = current_balance, current_balance = current_balance + $1 WHERE customer_code = $2",
		event.Data.Amount/100, event.Data.Customer.CustomerCode)

	if err != nil {
		tx.Rollback(context.Background())
		log.Printf("Failed to update balance: %v\n", err)
		return fmt.Errorf("failed to update balance: %w", err)
	}

	if err = tx.Commit(context.Background()); err != nil {
		log.Printf("Failed to commit transaction: %v\n", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Can we talk later this evening? I am kinda busy 
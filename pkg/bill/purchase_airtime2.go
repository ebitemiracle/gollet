package bill

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/smtp"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"go_code/database"
)

// AirtimePurchaseRequest represents the request payload for purchasing airtime
type AirtimePurchaseRequest struct {
	Amount      string `json:"amount"`
	Destination string `json:"destination"`
	UserID      int    `json:"user_id"`
}

// AirtimePurchaseResponse represents the response payload for a successful airtime purchase
type AirtimePurchaseResponse struct {
	Entity struct {
		Data []struct {
			Destination string `json:"destination"`
			Status      string `json:"status"`
		} `json:"data"`
		ReferenceID string `json:"reference_id"`
	} `json:"entity"`
}

// ErrorResponse represents the response payload for an error
type ErrorResponse struct {
	Error string `json:"error"`
}

// Response represents the generic response structure
type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result,omitempty"`
}

// BalanceResponse represents the response payload for the Dojah balance
type BalanceResponse struct {
	Entity struct {
		WalletBalance string `json:"wallet_balance"`
	} `json:"entity"`
	Error string `json:"error"`
}

// AirtimePurchaseHandler handles the airtime purchase process
func AirtimePurchaseHandler(c *gin.Context) {
	var purchaseRequest AirtimePurchaseRequest
	if err := c.ShouldBindJSON(&purchaseRequest); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Status:  "error",
			Message: "Invalid JSON input: " + err.Error(),
		})
		return
	}

	// Step 1: Validate the amount
	amount, err := strconv.ParseFloat(purchaseRequest.Amount, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Status:  "error",
			Message: "Invalid amount format: " + err.Error(),
		})
		return
	}

	// Step 2: Check the user's balance
	db, err := database.PostgreSQLConnect()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to connect to database: " + err.Error(),
		})
		return
	}
	defer db.Close(context.Background())

	var currentBalance float64
	err = db.QueryRow(context.Background(), "SELECT current_balance FROM wallet WHERE user_id = $1", purchaseRequest.UserID).Scan(&currentBalance)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to retrieve balance: " + err.Error(),
		})
		return
	}

	if currentBalance < amount {
		c.JSON(http.StatusBadRequest, Response{
			Status:  "error",
			Message: "Insufficient balance.",
			Result:  map[string]float64{"current_balance": currentBalance},
		})
		return
	}

	// Step 3: Check the Dojah balance
	dojahBalance, err := CheckDojahBalance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to check Dojah balance: " + err.Error(),
		})
		return
	}

	dojahBalanceFloat, err := strconv.ParseFloat(dojahBalance, 64)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to parse Dojah balance: " + err.Error(),
		})
		return
	}

	if dojahBalanceFloat < amount {
		SendInsufficientBalanceEmail()
		c.JSON(http.StatusServiceUnavailable, Response{
			Status:  "error",
			Message: "Insufficient balance in our vault, please try again later.",
		})
		return
	}

	// Step 4: Proceed with airtime purchase
	payloadJSON, err := json.Marshal(purchaseRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to marshal JSON: " + err.Error(),
		})
		return
	}

	url := "https://api.dojah.io/api/v1/purchase/airtime"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadJSON))
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to create request: " + err.Error(),
		})
		return
	}

	req.Header.Set("AppId", os.Getenv("DOJAH_APP_ID"))
	req.Header.Set("Authorization", os.Getenv("DOJAH_SECRET_KEY"))
	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to send request: " + err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to read response: " + err.Error(),
		})
		return
	}

	if resp.StatusCode != http.StatusOK {
		var errorResponse ErrorResponse
		if err := json.Unmarshal(body, &errorResponse); err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Status:  "error",
				Message: "Failed to parse error response: " + err.Error(),
				Result:  json.RawMessage(body),
			})
			return
		}
		c.JSON(http.StatusBadRequest, Response{
			Status:  "error",
			Message: errorResponse.Error,
		})
		return
	}

	var purchaseResponse AirtimePurchaseResponse
	if err := json.Unmarshal(body, &purchaseResponse); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to parse success response: " + err.Error(),
			Result:  json.RawMessage(body),
		})
		return
	}

	// Step 5: Update the user transaction database
	if err := SaveTransactionData(purchaseRequest.UserID, purchaseResponse.Entity.ReferenceID, amount, "debit"); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to save transaction data: " + err.Error(),
		})
		return
	}
	

	// Step 6: Update balance
	if err := UpdateBalanceforAirtime(purchaseRequest.UserID, amount); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to update balance: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Status:  "success",
		Message: "Airtime purchase successful",
		Result:  purchaseResponse.Entity,
	})
}

func CheckDojahBalance() (string, error) {
	url := "https://api.dojah.io/api/v1/balance"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("AppId", os.Getenv("DOJAH_APP_ID"))
	req.Header.Set("Authorization", os.Getenv("DOJAH_SECRET_KEY"))
	req.Header.Set("accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to retrieve balance: %s", string(body))
	}

	var balanceResponse BalanceResponse
	if err := json.Unmarshal(body, &balanceResponse); err != nil {
		return "", err
	}

	if balanceResponse.Error != "" {
		return "", fmt.Errorf("error from Dojah: %s", balanceResponse.Error)
	}

	return balanceResponse.Entity.WalletBalance, nil
}

func SendInsufficientBalanceEmail() {
	from := os.Getenv("SMTP_FROM")
	pass := os.Getenv("SMTP_PASS")
	to := os.Getenv("ADMIN_EMAIL")
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")

	msg := "Subject: Insufficient Balance Alert\n\nThe balance in the Dojah wallet is insufficient to process transactions. Please top up the wallet."

	auth := smtp.PlainAuth("", from, pass, smtpHost)
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{to}, []byte(msg))
	if err != nil {
		// Log the error
		fmt.Println("Failed to send email:", err)
	}
}

func SaveTransactionData(userID int, referenceID string, amount float64, transactionType string) error {
	db, err := database.PostgreSQLConnect()
	if err != nil {
		return err
	}
	defer db.Close(context.Background())

	query := `INSERT INTO user_transaction (user_id, reference, amount, transaction_type, narration) VALUES ($1, $2, $3, $4, $5)`
	_, err = db.Exec(context.Background(), query, userID, referenceID, amount, transactionType, "Airtime purchase")
	return err
}

func UpdateBalanceforAirtime(userID int, amount float64) error {
	db, err := database.PostgreSQLConnect()
	if err != nil {
		return err
	}
	defer db.Close(context.Background())

	query := `UPDATE wallet SET previous_balance = current_balance, current_balance = current_balance - $1 WHERE user_id = $2`
	_, err = db.Exec(context.Background(), query, amount, userID)
	return err
}
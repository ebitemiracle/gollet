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

// DataPurchaseRequest represents the request payload for purchasing data
type DataPurchaseRequest struct {
	Destination string `json:"destination"`
	Plan        string `json:"plan"`
	UserID      int    `json:"user_id"`
}

// DataPurchaseResponse represents the response payload for a successful data purchase
type DataPurchaseResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  struct {
		Data []struct {
			Destination string `json:"destination"`
			Status      string `json:"status"`
		} `json:"data"`
		ReferenceID string `json:"reference_id"`
	} `json:"result"`
}

// ErrorResponse represents the response payload for an error
type ErrorResponsee struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// // Response represents the generic response structure
type Data2Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result,omitempty"`
}

// // BalanceResponse represents the response payload for the Dojah balance
type Data2BalanceResponse struct {
	Entity struct {
		WalletBalance string `json:"wallet_balance"`
	} `json:"entity"`
	Error string `json:"error"`
}

// DataPlansResponse represents the response payload for available data plans
type DataPlansResponse struct {
	Entity []struct {
		Amount      int64  `json:"amount"`
		Plan        string `json:"plan"`
		Description string `json:"description"`
	} `json:"entity"`
}

// DataPurchaseHandler handles the data purchase process
func DataPurchaseHandler(c *gin.Context) {
	var purchaseRequest DataPurchaseRequest
	if err := c.ShouldBindJSON(&purchaseRequest); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Status:  "error",
			Message: "Invalid JSON input: " + err.Error(),
		})
		return
	}

	// Step 1: Fetch available data plans
	dataPlans, err := fetchDataPlans()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to fetch data plans: " + err.Error(),
		})
		return
	}

	// Step 2: Check if the requested plan is available
	var planCost float64
	planFound := false
	for _, plan := range dataPlans.Entity {
		if plan.Plan == purchaseRequest.Plan {
			planCost = float64(plan.Amount)
			planFound = true
			break
		}
	}

	if !planFound {
		c.JSON(http.StatusBadRequest, Response{
			Status:  "error",
			Message: "Invalid plan selected",
		})
		return
	}

	// Step 3: Check the user's balance
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

	if currentBalance < planCost {
		c.JSON(http.StatusBadRequest, Response{
			Status:  "error",
			Message: "Insufficient balance.",
			Result:  map[string]float64{"current_balance": currentBalance},
		})
		return
	}

	// Step 4: Check the Dojah balance
	dojahBalance, err := checkDojahBalance()
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

	if dojahBalanceFloat < planCost {
		sendInsufficientBalanceEmail()
		c.JSON(http.StatusServiceUnavailable, Response{
			Status:  "error",
			Message: "Insufficient balance in our vault, please try again later.",
		})
		return
	}

	// Step 5: Proceed with data purchase
	payloadJSON, err := json.Marshal(purchaseRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to marshal JSON: " + err.Error(),
		})
		return
	}

	url := "https://api.dojah.io/api/v1/purchase/data"
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
		var errorResponse ErrorResponsee
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
			Message: errorResponse.Message,
		})
		return
	}

	var purchaseResponse DataPurchaseResponse
	if err := json.Unmarshal(body, &purchaseResponse); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to parse success response: " + err.Error(),
			Result:  json.RawMessage(body),
		})
		return
	}

	// Step 6: Update the user transaction database
	if err := SaveTransactionDataforData(purchaseRequest.UserID, purchaseResponse.Result.ReferenceID, planCost, "debit"); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to save transaction data: " + err.Error(),
		})
		return
	}

	// Step 7: Update balance
	if err := UpdateBalance(purchaseRequest.UserID, planCost); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to update balance: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Status:  "success",
		Message: "Data purchase successful",
		Result:  purchaseResponse.Result,
	})
}

func fetchDataPlans() (*DataPlansResponse, error) {
	url := "https://api.dojah.io/api/v1/purchase/data/plans"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("AppId", os.Getenv("DOJAH_APP_ID"))
	req.Header.Set("Authorization", os.Getenv("DOJAH_SECRET_KEY"))
	req.Header.Set("accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to retrieve data plans: %s", string(body))
	}

	var dataPlansResponse DataPlansResponse
	if err := json.Unmarshal(body, &dataPlansResponse); err != nil {
		return nil, err
	}

	return &dataPlansResponse, nil
}

func checkDojahBalance() (string, error) {
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

	var balanceResponse Data2BalanceResponse
	if err := json.Unmarshal(body, &balanceResponse); err != nil {
		return "", err
	}

	if balanceResponse.Error != "" {
		return "", fmt.Errorf("error from Dojah: %s", balanceResponse.Error)
	}

	return balanceResponse.Entity.WalletBalance, nil
}

func sendInsufficientBalanceEmail() {
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

func SaveTransactionDataforData(userID int, referenceID string, planCost float64, transactionType string) error {
	db, err := database.PostgreSQLConnect()
	if err != nil {
		return err
	}
	defer db.Close(context.Background())

	query := `INSERT INTO user_transaction (user_id, reference_id, amount, transaction_type, narration) VALUES ($1, $2, $3, $4, $5)`
	_, err = db.Exec(context.Background(), query, userID, referenceID, planCost, transactionType, "Data purchase")
	return err
}

func UpdateBalance(userID int, planCost float64) error {
	db, err := database.PostgreSQLConnect()
	if err != nil {
		return err
	}
	defer db.Close(context.Background())

	query := `UPDATE wallet SET previous_balance = current_balance, current_balance = current_balance - $1 WHERE user_id = $2`
	_, err = db.Exec(context.Background(), query, planCost, userID)
	return err
}
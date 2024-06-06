package transaction

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"go_code/database"
)

// FundTransfer represents the request payload for fund transfer
type FundTransfer struct {
	AccountNumber string  `json:"account_number"`
	BankCode      string  `json:"bank_code"`
	UserID        int     `json:"user_id"`
	Source        string  `json:"source"`
	Reason        string  `json:"reason"`
	Amount        float64 `json:"amount"` // Amount in Naira
}

// Response represents the generic response structure
type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result,omitempty"`
}

// ResolveBankAccountResponse represents the response from the Paystack API for resolving a bank account
type ResolveBankAccountResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		AccountNumber string `json:"account_number"`
		AccountName   string `json:"account_name"`
	} `json:"data"`
}

// TransferRecipientResponse represents the response from the Paystack API for creating a transfer recipient
type TransferRecipientResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		RecipientCode string `json:"recipient_code"`
	} `json:"data"`
}

// InitiateTransfer represents the payload for initiating a transfer
type InitiateTransfer struct {
	UserID    int    `json:"user_id"`
	Source    string `json:"source"`
	Reason    string `json:"reason"`
	Amount    float64  `json:"amount"`
	Recipient string `json:"recipient"`
}

// InitiateTransferResponse represents the response from the Paystack API for initiating a transfer
type InitiateTransferResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		TransferCode       string          `json:"transfer_code"`
		TransferReference  string          `json:"reference"`
		Recipient          json.RawMessage `json:"recipient"` // Keeping as json.RawMessage to parse dynamically later
	} `json:"data"`
}

// VerifyTransferResponse represents the response from the Paystack API for verifying a transfer
type VerifyTransferResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Status string `json:"status"`
	} `json:"data"`
}

// FundTransferHandler handles the complete fund transfer process
func FundTransferHandler(c *gin.Context) {
	var fundTransfer FundTransfer
	if err := c.ShouldBindJSON(&fundTransfer); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Status:  "error",
			Message: "Invalid JSON input: " + err.Error(),
		})
		return
	}

	// Check the available balance
	if err := checkBalanceAndProceed(c, fundTransfer); err != nil {
		return
	}

	// Step 1: Resolve the bank account information
	resolveURL := fmt.Sprintf("https://api.paystack.co/bank/resolve?account_number=%s&bank_code=%s", fundTransfer.AccountNumber, fundTransfer.BankCode)
	authorization := "Bearer " + os.Getenv("PAYSTACK_SECRET_KEY")

	req, err := http.NewRequest("GET", resolveURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to create request: " + err.Error(),
		})
		return
	}
	req.Header.Set("Authorization", authorization)

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

	var resolveResponse ResolveBankAccountResponse
	if err := json.Unmarshal(body, &resolveResponse); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to parse response: " + err.Error(),
			Result:  json.RawMessage(body),
		})
		return
	}

	if !resolveResponse.Status {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to resolve bank account: " + resolveResponse.Message,
			Result:  resolveResponse,
		})
		return
	}

	accountName := resolveResponse.Data.AccountName
	bank := fundTransfer.BankCode

	// Step 2: Use the resolved information to create a transfer recipient
	recipientCreateURL := "https://api.paystack.co/transferrecipient"
	recipientData := map[string]string{
		"type":           "nuban",
		"name":           resolveResponse.Data.AccountName,
		"account_number": resolveResponse.Data.AccountNumber,
		"bank_code":      fundTransfer.BankCode,
		"currency":       "NGN",
	}
	recipientDataJSON, err := json.Marshal(recipientData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to marshal JSON: " + err.Error(),
		})
		return
	}

	req, err = http.NewRequest("POST", recipientCreateURL, bytes.NewBuffer(recipientDataJSON))
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to create request: " + err.Error(),
		})
		return
	}
	req.Header.Set("Authorization", authorization)
	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to send request: " + err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to read response: " + err.Error(),
		})
		return
	}

	var transferRecipientResponse TransferRecipientResponse
	if err := json.Unmarshal(body, &transferRecipientResponse); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to parse response: " + err.Error(),
			Result:  json.RawMessage(body),
		})
		return
	}

	if !transferRecipientResponse.Status {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to create transfer recipient: " + transferRecipientResponse.Message,
			Result:  transferRecipientResponse,
		})
		return
	}

	recipientCode := transferRecipientResponse.Data.RecipientCode

	// Step 3: Initiate the transfer
	initiateTransferURL := "https://api.paystack.co/transfer"
	transferData := InitiateTransfer{
		UserID:    fundTransfer.UserID,
		Source:    fundTransfer.Source,
		Reason:    fundTransfer.Reason,
		Amount:    fundTransfer.Amount * 100.00, // Convert Naira to Kobo
		Recipient: recipientCode,
	}
	transferDataJSON, err := json.Marshal(transferData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to marshal JSON: " + err.Error(),
		})
		return
	}

	req, err = http.NewRequest("POST", initiateTransferURL, bytes.NewBuffer(transferDataJSON))
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to create request: " + err.Error(),
		})
		return
	}
	req.Header.Set("Authorization", authorization)
	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to send request: " + err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to read response: " + err.Error(),
		})
		return
	}

	var initiateTransferResponse InitiateTransferResponse
	if err := json.Unmarshal(body, &initiateTransferResponse); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to parse response: " + err.Error(),
			Result:  json.RawMessage(body),
		})
		return
	}

	if !initiateTransferResponse.Status {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to initiate transfer: " + initiateTransferResponse.Message,
			Result:  initiateTransferResponse,
		})
		return
	}

	// Save the transfer data in the database
	transferCode := initiateTransferResponse.Data.TransferCode
	if err := saveTransferDataInDatabase(fundTransfer.UserID, recipientCode, transferCode, accountName, bank); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to save transfer data: " + err.Error(),
		})
		return
	}

	finalResult := map[string]interface{}{
		"verify_transfer_response": initiateTransferResponse.Data,
		// "recipient_details":        recipientDetails,
	}

	finalStatus := "success"
	if !initiateTransferResponse.Status {
		finalStatus = "error"
	}

	c.JSON(http.StatusOK, Response{
		Status:  finalStatus,
		Message: "Fund transfer process completed",
		Result:  finalResult,
	})
}

// checkBalanceAndProceed checks the user's balance and proceeds with the transfer if sufficient
func checkBalanceAndProceed(c *gin.Context, fundTransfer FundTransfer) error {
	db, err := database.PostgreSQLConnect()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to connect to database: " + err.Error(),
		})
		return err
	}
	defer db.Close(context.Background())

	var currentBalance float64
	err = db.QueryRow(context.Background(), "SELECT current_balance FROM wallet WHERE user_id = $1", fundTransfer.UserID).Scan(&currentBalance)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to retrieve balance: " + err.Error(),
		})
		return err
	}

	if currentBalance < fundTransfer.Amount {
		c.JSON(http.StatusBadRequest, Response{
			Status:  "error",
			Message: "Insufficient balance.",
			Result:  map[string]float64{"current_balance": currentBalance},
		})
		return fmt.Errorf("insufficient balance")
	}

	return nil
}

// saveTransferDataInDatabase saves transfer data in the database
func saveTransferDataInDatabase(userID int, recipientCode, transferCode, accountName, bank string) error {
	db, err := database.PostgreSQLConnect()
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close(context.Background())

	_, err = db.Exec(context.Background(),
		"INSERT INTO user_transaction (user_id, recipient_code, transfer_code, transaction_type, account_name, bank) VALUES ($1, $2, $3, $4, $5, $6)",
		userID, recipientCode, transferCode, "debit", accountName, bank)
	if err != nil {
		return fmt.Errorf("failed to insert transfer data: %w", err)
	}

	return nil
}
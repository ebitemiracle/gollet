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
	UserID    int     `json:"user_id"`
	Source    string  `json:"source"`
	Reason    string  `json:"reason"`
	Amount    float64 `json:"amount"`
	Recipient string  `json:"recipient"`
}

// InitiateTransferResponse represents the response from the Paystack API for initiating a transfer
type InitiateTransferResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		TransferCode      string      `json:"transfer_code"`
		TransferReference string      `json:"reference"`
		Recipient         interface{} `json:"recipient"` // Changed to interface{} to handle different types
	} `json:"data"`
}

// VerifyTransferResponse represents the response from the Paystack API for verifying a transfer
type VerifyTransferResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Status    string `json:"status"`
		Recipient struct {
			Details struct {
				BankName string `json:"bank_name"`
			} `json:"details"`
		} `json:"recipient"`
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

	// Resolve the bank account information
	accountName, err := resolveBankAccount(fundTransfer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	// Create transfer recipient
	recipientCode, err := createTransferRecipient(fundTransfer, accountName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	// Initiate the transfer
	transferCode, reference, err := initiateTransfer(fundTransfer, recipientCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	// Verify the transfer
	bankName, err := verifyTransfer(reference)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to verify transfer: " + err.Error(),
		})
		return
	}

	// Save the transfer data in the database
	if err := saveTransferDataInDatabase(fundTransfer.UserID, recipientCode, transferCode, accountName, bankName, fundTransfer.BankCode); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to save transfer data: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Status:  "success",
		Message: "Fund transfer process completed and verified",
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

// resolveBankAccount resolves the bank account information using the Paystack API
func resolveBankAccount(fundTransfer FundTransfer) (string, error) {
	resolveURL := fmt.Sprintf("https://api.paystack.co/bank/resolve?account_number=%s&bank_code=%s", fundTransfer.AccountNumber, fundTransfer.BankCode)
	authorization := "Bearer " + os.Getenv("PAYSTACK_SECRET_KEY")

	req, err := http.NewRequest("GET", resolveURL, nil)
	if err != nil {
		return "", fmt.Errorf("Failed to create request: %w", err)
	}
	req.Header.Set("Authorization", authorization)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Failed to read response: %w", err)
	}

	var resolveResponse ResolveBankAccountResponse
	if err := json.Unmarshal(body, &resolveResponse); err != nil {
		return "", fmt.Errorf("Failed to parse response: %w", err)
	}

	if !resolveResponse.Status {
		return "", fmt.Errorf("Failed to resolve bank account: %s", resolveResponse.Message)
	}

	return resolveResponse.Data.AccountName, nil
}

// createTransferRecipient creates a transfer recipient using the Paystack API
func createTransferRecipient(fundTransfer FundTransfer, accountName string) (string, error) {
	recipientCreateURL := "https://api.paystack.co/transferrecipient"
	recipientData := map[string]string{
		"type":           "nuban",
		"name":           accountName,
		"account_number": fundTransfer.AccountNumber,
		"bank_code":      fundTransfer.BankCode,
		"currency":       "NGN",
	}
	recipientDataJSON, err := json.Marshal(recipientData)
	if err != nil {
		return "", fmt.Errorf("Failed to marshal JSON: %w", err)
	}

	authorization := "Bearer " + os.Getenv("PAYSTACK_SECRET_KEY")
	req, err := http.NewRequest("POST", recipientCreateURL, bytes.NewBuffer(recipientDataJSON))
	if err != nil {
		return "", fmt.Errorf("Failed to create request: %w", err)
	}
	req.Header.Set("Authorization", authorization)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Failed to read response: %w", err)
	}

	var transferRecipientResponse TransferRecipientResponse
	if err := json.Unmarshal(body, &transferRecipientResponse); err != nil {
		return "", fmt.Errorf("Failed to parse response: %w", err)
	}

	if !transferRecipientResponse.Status {
		return "", fmt.Errorf("Failed to create transfer recipient: %s", transferRecipientResponse.Message)
	}

	return transferRecipientResponse.Data.RecipientCode, nil
}

// initiateTransfer initiates the transfer using the Paystack API
func initiateTransfer(fundTransfer FundTransfer, recipientCode string) (string, string, error) {
	initiateTransferURL := "https://api.paystack.co/transfer"
	transferData := struct {
		Source    string  `json:"source"`
		Reason    string  `json:"reason"`
		Amount    float64 `json:"amount"`
		Recipient string  `json:"recipient"`
	}{
		Source:    fundTransfer.Source,
		Reason:    fundTransfer.Reason,
		Amount:    fundTransfer.Amount * 100, // Convert Naira to Kobo
		Recipient: recipientCode,
	}
	transferDataJSON, err := json.Marshal(transferData)
	if err != nil {
		return "", "", fmt.Errorf("Failed to marshal JSON: %w", err)
	}

	authorization := "Bearer " + os.Getenv("PAYSTACK_SECRET_KEY")
	req, err := http.NewRequest("POST", initiateTransferURL, bytes.NewBuffer(transferDataJSON))
	if err != nil {
		return "", "", fmt.Errorf("Failed to create request: %w", err)
	}
	req.Header.Set("Authorization", authorization)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("Failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("Failed to read response: %w", err)
	}

	// Log the actual response body for debugging
	fmt.Println("Response Body:", string(body))

	var initiateTransferResponse InitiateTransferResponse
	if err := json.Unmarshal(body, &initiateTransferResponse); err != nil {
		return "", "", fmt.Errorf("Failed to parse response: %w", err)
	}

	if !initiateTransferResponse.Status {
		return "", "", fmt.Errorf("Failed to initiate transfer: %s", initiateTransferResponse.Message)
	}

	// Handle the recipient data based on its type
	switch recipient := initiateTransferResponse.Data.Recipient.(type) {
	case float64:
		// Recipient is an ID
		fmt.Printf("Recipient ID: %v\n", recipient)
	case map[string]interface{}:
		// Recipient is a detailed object
		fmt.Printf("Recipient Data: %+v\n", recipient)
	default:
		return "", "", fmt.Errorf("Unexpected recipient type: %T", recipient)
	}

	return initiateTransferResponse.Data.TransferCode, initiateTransferResponse.Data.TransferReference, nil
}

// verifyTransfer verifies the transfer using the Paystack API
func verifyTransfer(reference string) (string, error) {
	verifyTransferURL := fmt.Sprintf("https://api.paystack.co/transfer/verify/%s", reference)
	authorization := "Bearer " + os.Getenv("PAYSTACK_SECRET_KEY")

	req, err := http.NewRequest("GET", verifyTransferURL, nil)
	if err != nil {
		return "", fmt.Errorf("Failed to create request: %w", err)
	}
	req.Header.Set("Authorization", authorization)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Failed to read response: %w", err)
	}

	var verifyTransferResponse VerifyTransferResponse
	if err := json.Unmarshal(body, &verifyTransferResponse); err != nil {
		return "", fmt.Errorf("Failed to parse response: %w", err)
	}

	if !verifyTransferResponse.Status {
		return "", fmt.Errorf("Failed to verify transfer: %s", verifyTransferResponse.Message)
	}

	return verifyTransferResponse.Data.Recipient.Details.BankName, nil
}

// saveTransferDataInDatabase saves the transfer data in the database
func saveTransferDataInDatabase(userID int, recipientCode, transferCode, accountName, bankName, bankCode string) error {
	db, err := database.PostgreSQLConnect()
	if err != nil {
		return fmt.Errorf("Failed to connect to database: %w", err)
	}
	defer db.Close(context.Background())

	sqlStatement := `
		INSERT INTO user_transaction (user_id, recipient_code, transfer_code, account_name, bank_name, bank_code, transaction_type)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err = db.Exec(context.Background(), sqlStatement, userID, recipientCode, transferCode, accountName, bankName, bankCode, "debit")
	if err != nil {
		return fmt.Errorf("Failed to save transfer data: %w", err)
	}

	return nil
}
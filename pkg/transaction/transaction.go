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
	AccountNumber string `json:"account_number"`
	BankCode      string `json:"bank_code"`
	UserID        int    `json:"user_id"`
	Source        string `json:"source"`
	Reason        string `json:"reason"`
	Amount        int64  `json:"amount"` // Amount in Naira
}

// ResolveBankAccountResponse represents the response payload for resolving a bank account
type ResolveBankAccountResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		AccountNumber string `json:"account_number"`
		AccountName   string `json:"account_name"`
		BankID        int    `json:"bank_id"`
		BankName      string `json:"bank_name"`
	} `json:"data"`
}

// TransferRecipientResponse represents the response payload for creating a transfer recipient
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
	Amount    int64  `json:"amount"` // Amount in Kobo
	Recipient string `json:"recipient"`
}

// InitiateTransferResponse represents the response payload for initiating a transfer
type InitiateTransferResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		TransferCode      string `json:"transfer_code"`
		TransferReference string `json:"reference"`
		Status            string `json:"status"`
		Recipient         json.RawMessage `json:"recipient"`
	} `json:"data"`
}

// VerifyTransferResponse represents the response payload for verifying a transfer
type VerifyTransferResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Status            string `json:"status"`
		TransferCode      string `json:"transfer_code"`
		TransferReference string `json:"reference"`
		Recipient         struct {
			Details struct {
				AccountNumber string `json:"account_number"`
				AccountName   string `json:"account_name"`
				BankCode      string `json:"bank_code"`
				BankName      string `json:"bank_name"`
			} `json:"details"`
		} `json:"recipient"`
	} `json:"data"`
}

// Response represents the generic response structure
type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result,omitempty"`
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
		Amount:    fundTransfer.Amount * 100, // Convert amount to Kobo
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

	// Save the user_id, recipient_code, and transfer_code in the database
	if err := saveTransferDataInDatabase(fundTransfer.UserID, recipientCode, initiateTransferResponse.Data.TransferCode); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to save transfer data: " + err.Error(),
		})
		return
	}

	// Step 4: Verify the transfer
	verifyURL := fmt.Sprintf("https://api.paystack.co/transfer/verify/%s", initiateTransferResponse.Data.TransferReference)
	req, err = http.NewRequest("GET", verifyURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to create request: " + err.Error(),
		})
		return
	}
	req.Header.Set("Authorization", authorization)

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

	var verifyTransferResponse VerifyTransferResponse
	if err := json.Unmarshal(body, &verifyTransferResponse); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to parse response: " + err.Error(),
			Result:  json.RawMessage(body),
		})
		return
	}

	if !verifyTransferResponse.Status {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to verify transfer",
			Result:  verifyTransferResponse,
		})
		return
	}

	// Handle the dynamic Recipient field
	var recipientDetails interface{}
	if err := json.Unmarshal(initiateTransferResponse.Data.Recipient, &recipientDetails); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to parse recipient details: " + err.Error(),
			Result:  initiateTransferResponse.Data.Recipient,
		})
		return
	}

	finalResult := map[string]interface{}{
		"verify_transfer_response": verifyTransferResponse.Data,
		"recipient_details":        recipientDetails,
	}

	finalStatus := "success"
	if !initiateTransferResponse.Status || !verifyTransferResponse.Status {
		finalStatus = "error"
	}

	c.JSON(http.StatusOK, Response{
		Status:  finalStatus,
		Message: "Fund transfer process completed",
		Result:  finalResult,
	})
}

// saveTransferDataInDatabase saves transfer data in the database
func saveTransferDataInDatabase(userID int, recipientCode, transferCode string) error {
	db, err := database.PostgreSQLConnect()
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close(context.Background())

	_, err = db.Exec(context.Background(),
		"INSERT INTO user_transaction (user_id, recipient_code, transfer_code) VALUES ($1, $2, $3)",
		userID, recipientCode, transferCode)
	if err != nil {
		return fmt.Errorf("failed to insert transfer data: %w", err)
	}
	return nil
}




// CustomerResponse represents the response payload for fetching a customer
type CustomerResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Transactions           []interface{} `json:"transactions"`
		Subscriptions          []interface{} `json:"subscriptions"`
		Authorizations         []struct {
			AuthorizationCode string      `json:"authorization_code"`
			Bin               string      `json:"bin"`
			Last4             string      `json:"last4"`
			ExpMonth          string      `json:"exp_month"`
			ExpYear           string      `json:"exp_year"`
			Channel           string      `json:"channel"`
			CardType          string      `json:"card_type"`
			Bank              string      `json:"bank"`
			CountryCode       string      `json:"country_code"`
			Brand             string      `json:"brand"`
			Reusable          bool        `json:"reusable"`
			Signature         string      `json:"signature"`
			AccountName       interface{} `json:"account_name"`
		} `json:"authorizations"`
		FirstName              interface{}   `json:"first_name"`
		LastName               interface{}   `json:"last_name"`
		Email                  string        `json:"email"`
		Phone                  interface{}   `json:"phone"`
		Metadata               interface{}   `json:"metadata"`
		Domain                 string        `json:"domain"`
		CustomerCode           string        `json:"customer_code"`
		RiskAction             string        `json:"risk_action"`
		ID                     int           `json:"id"`
		Integration            int           `json:"integration"`
		CreatedAt              string        `json:"createdAt"`
		UpdatedAt              string        `json:"updatedAt"`
		TotalTransactions      int           `json:"total_transactions"`
		TotalTransactionValue  []interface{} `json:"total_transaction_value"`
		DedicatedAccount       interface{}   `json:"dedicated_account"`
		Identified             bool          `json:"identified"`
		Identifications        interface{}   `json:"identifications"`
	} `json:"data"`
}

// Response represents the generic response structure
// type Response struct {
// 	Status  string      `json:"status"`
// 	Message string      `json:"message"`
// 	Result  interface{} `json:"result,omitempty"`
// }

// GetCustomerHandler handles the request to fetch customer details
func GetCustomerHandler(c *gin.Context) {
	emailOrCode := c.Param("emailOrCode")
	url := fmt.Sprintf("https://api.paystack.co/customer/%s", emailOrCode)
	authorization := "Bearer " + os.Getenv("PAYSTACK_SECRET_KEY")

	req, err := http.NewRequest("GET", url, nil)
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

	var customerResponse CustomerResponse
	if err := json.Unmarshal(body, &customerResponse); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to parse response: " + err.Error(),
			Result:  json.RawMessage(body),
		})
		return
	}

	if !customerResponse.Status {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to fetch customer details: " + customerResponse.Message,
			Result:  customerResponse,
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Status:  "success",
		Message: "Customer details retrieved successfully",
		Result:  customerResponse.Data,
	})
}


// DedicatedAccountResponse represents the response payload for fetching a dedicated account

// DedicatedAccountResponse represents the response payload for fetching a dedicated account
type DedicatedAccountResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Transactions           []interface{} `json:"transactions"`
		Subscriptions          []interface{} `json:"subscriptions"`
		Authorizations         []interface{} `json:"authorizations"`
		FirstName              interface{}   `json:"first_name"`
		LastName               interface{}   `json:"last_name"`
		Email                  string        `json:"email"`
		Phone                  interface{}   `json:"phone"`
		Metadata               interface{}   `json:"metadata"`
		Domain                 string        `json:"domain"`
		CustomerCode           string        `json:"customer_code"`
		RiskAction             string        `json:"risk_action"`
		ID                     int           `json:"id"`
		Integration            int           `json:"integration"`
		CreatedAt              string        `json:"createdAt"`
		UpdatedAt              string        `json:"updatedAt"`
		TotalTransactions      int           `json:"total_transactions"`
		TotalTransactionValue  []interface{} `json:"total_transaction_value"`
		DedicatedAccount       struct {
			ID             int    `json:"id"`
			AccountName    string `json:"account_name"`
			AccountNumber  string `json:"account_number"`
			CreatedAt      string `json:"created_at"`
			UpdatedAt      string `json:"updated_at"`
			Currency       string `json:"currency"`
			Active         bool   `json:"active"`
			Assigned       bool   `json:"assigned"`
			Provider       struct {
				ID           int    `json:"id"`
				ProviderSlug string `json:"provider_slug"`
				BankID       int    `json:"bank_id"`
				BankName     string `json:"bank_name"`
			} `json:"provider"`
			Assignment struct {
				AssigneeID   int    `json:"assignee_id"`
				AssigneeType string `json:"assignee_type"`
				AccountType  string `json:"account_type"`
				Integration  int    `json:"integration"`
			} `json:"assignment"`
		} `json:"dedicated_account"`
	} `json:"data"`
}


// GetDedicatedAccountHandler handles the request to fetch dedicated account details
func GetDedicatedAccountHandler(c *gin.Context) {
	dedicatedAccountId := c.Param("dedicatedAccountId")
	url := fmt.Sprintf("https://api.paystack.co/dedicated_account/%s", dedicatedAccountId)
	authorization := "Bearer " + os.Getenv("PAYSTACK_SECRET_KEY")

	req, err := http.NewRequest("GET", url, nil)
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

	fmt.Printf("Raw response: %s\n", string(body)) // Debug log the raw response

	var dedicatedAccountResponse DedicatedAccountResponse
	if err := json.Unmarshal(body, &dedicatedAccountResponse); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to parse response: " + err.Error(),
			Result:  json.RawMessage(body),
		})
		return
	}

	if !dedicatedAccountResponse.Status {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to fetch dedicated account details: " + dedicatedAccountResponse.Message,
			Result:  dedicatedAccountResponse,
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Status:  "success",
		Message: "Dedicated account details retrieved successfully",
		Result:  dedicatedAccountResponse.Data,
	})
}

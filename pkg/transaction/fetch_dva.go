package transaction

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

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

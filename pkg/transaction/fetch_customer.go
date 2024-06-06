package transaction

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

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

package third_party_balance

import (
	"encoding/json"
	// "fmt"
	"io/ioutil"
	// "log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// BalanceResponse represents the response payload for the Dojah balance
type BalanceResponse struct {
	Entity struct {
		WalletBalance string `json:"wallet_balance"`
	} `json:"entity"`
	Error string `json:"error"`
}

// Response represents the generic response structure
type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result,omitempty"`
}

// BalanceHandler handles the request to check the Dojah balance
func BalanceHandler(c *gin.Context) {
	// Prepare the request
	url := "https://api.dojah.io/api/v1/balance"
	req, err := http.NewRequest("GET", url, nil)
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
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to retrieve balance",
			Result:  string(body),
		})
		return
	}

	var balanceResponse BalanceResponse
	if err := json.Unmarshal(body, &balanceResponse); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to parse response: " + err.Error(),
			Result:  json.RawMessage(body),
		})
		return
	}

	if balanceResponse.Error != "" {
		c.JSON(http.StatusBadRequest, Response{
			Status:  "error",
			Message: balanceResponse.Error,
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Status:  "success",
		Message: "Balance retrieved successfully",
		Result:  balanceResponse.Entity,
	})
}

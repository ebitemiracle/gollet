package wallet

import (
	// "context"
	"encoding/json"
	// "go_code/database"
	"io/ioutil"
	"net/http"
	"os"
	// "strconv"

	"github.com/gin-gonic/gin"
)

// Bank represents the bank data structure
type Bank struct {
    Name        string `json:"name"`
    Slug        string `json:"slug"`
    Code        string `json:"code"`
    Longcode    string `json:"longcode"`
    Gateway     string `json:"gateway,omitempty"`
    PayWithBank bool   `json:"pay_with_bank"`
    Active      bool   `json:"active"`
    IsDeleted   bool   `json:"is_deleted"`
    Country     string `json:"country"`
    Currency    string `json:"currency"`
    Type        string `json:"type"`
    ID          int    `json:"id"`
    CreatedAt   string `json:"createdAt"`
    UpdatedAt   string `json:"updatedAt"`
}

// PaystackBankResponse represents the response from the Paystack API
type PaystackBankResponse struct {
    Status  bool   `json:"status"`
    Message string `json:"message"`
    Data    []Bank `json:"data"`
    Meta    struct {
        Next     string `json:"next"`
        Previous string `json:"previous"`
        PerPage  int    `json:"perPage"`
    } `json:"meta"`
}

// ViewAllBanksHandler handles the request to view all banks
func ViewAllBanksHandler(c *gin.Context) {
    url := "https://api.paystack.co/bank"
    authorization := "Bearer " + os.Getenv("PAYSTACK_SECRET_KEY")

    client := &http.Client{}
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        c.JSON(http.StatusInternalServerError, Response{
            Status:  "error",
            Message: "Failed to create request: " + err.Error(),
        })
        return
    }
    req.Header.Set("Authorization", authorization)

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
            Message: "Failed to retrieve banks: " + string(body),
        })
        return
    }

    var paystackResponse PaystackBankResponse
    if err := json.Unmarshal(body, &paystackResponse); err != nil {
        c.JSON(http.StatusInternalServerError, Response{
            Status:  "error",
            Message: "Failed to parse response: " + err.Error(),
        })
        return
    }

    if !paystackResponse.Status {
        c.JSON(http.StatusInternalServerError, Response{
            Status:  "error",
            Message: "Failed to retrieve banks: " + paystackResponse.Message,
        })
        return
    }

    c.JSON(http.StatusOK, Response{
        Status:  "success",
        Message: "Banks retrieved successfully",
        Result:  paystackResponse.Data,
    })
}


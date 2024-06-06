package bill

// import (
// 	"bytes"
// 	"encoding/json"
// 	"io/ioutil"
// 	"net/http"
// 	"os"

// 	"github.com/gin-gonic/gin"
// )

// // AirtimePurchaseRequest represents the request payload for purchasing airtime
// // type AirtimePurchaseRequest struct {
// // 	Amount      string `json:"amount"`
// // 	Destination string `json:"destination"`
// // }

// // // AirtimePurchaseResponse represents the response payload for a successful airtime purchase
// // type AirtimePurchaseResponse struct {
// // 	Entity struct {
// // 		Data []struct {
// // 			Destination string `json:"destination"`
// // 			Status      string `json:"status"`
// // 		} `json:"data"`
// // 		ReferenceID string `json:"reference_id"`
// // 	} `json:"entity"`
// // }

// // // ErrorResponse represents the response payload for an error
// // type ErrorResponse struct {
// // 	Error string `json:"error"`
// // }

// // Response represents the generic response structure
// // type Response struct {
// // 	Status  string      `json:"status"`
// // 	Message string      `json:"message"`
// // 	Result  interface{} `json:"result,omitempty"`
// // }

// // AirtimePurchaseHandler handles the airtime purchase process
// func AirtimePurchaseHandler(c *gin.Context) {
// 	var purchaseRequest AirtimePurchaseRequest
// 	if err := c.ShouldBindJSON(&purchaseRequest); err != nil {
// 		c.JSON(http.StatusBadRequest, Response{
// 			Status:  "error",
// 			Message: "Invalid JSON input: " + err.Error(),
// 		})
// 		return
// 	}

// 	// Create the request payload
// 	payloadJSON, err := json.Marshal(purchaseRequest)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, Response{
// 			Status:  "error",
// 			Message: "Failed to marshal JSON: " + err.Error(),
// 		})
// 		return
// 	}

// 	// Prepare the request
// 	url := "https://api.dojah.io/api/v1/purchase/airtime"
// 	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadJSON))
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, Response{
// 			Status:  "error",
// 			Message: "Failed to create request: " + err.Error(),
// 		})
// 		return
// 	}

// 	req.Header.Set("AppId", os.Getenv("DOJAH_APP_ID"))
// 	req.Header.Set("Authorization", os.Getenv("DOJAH_SECRET_KEY"))
// 	req.Header.Set("accept", "application/json")
// 	req.Header.Set("content-type", "application/json")

// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, Response{
// 			Status:  "error",
// 			Message: "Failed to send request: " + err.Error(),
// 		})
// 		return
// 	}
// 	defer resp.Body.Close()

// 	body, err := ioutil.ReadAll(resp.Body)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, Response{
// 			Status:  "error",
// 			Message: "Failed to read response: " + err.Error(),
// 		})
// 		return
// 	}

// 	// Check the response status
// 	if resp.StatusCode != http.StatusOK {
// 		var errorResponse ErrorResponse
// 		if err := json.Unmarshal(body, &errorResponse); err != nil {
// 			c.JSON(http.StatusInternalServerError, Response{
// 				Status:  "error",
// 				Message: "Failed to parse error response: " + err.Error(),
// 				Result:  json.RawMessage(body),
// 			})
// 			return
// 		}
// 		c.JSON(http.StatusBadRequest, Response{
// 			Status:  "error",
// 			Message: errorResponse.Error,
// 		})
// 		return
// 	}

// 	var purchaseResponse AirtimePurchaseResponse
// 	if err := json.Unmarshal(body, &purchaseResponse); err != nil {
// 		c.JSON(http.StatusInternalServerError, Response{
// 			Status:  "error",
// 			Message: "Failed to parse success response: " + err.Error(),
// 			Result:  json.RawMessage(body),
// 		})
// 		return
// 	}

// 	c.JSON(http.StatusOK, Response{
// 		Status:  "success",
// 		Message: "Airtime purchase successful",
// 		Result:  purchaseResponse.Entity,
// 	})
// }
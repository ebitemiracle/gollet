package bill

// import (
// 	"bytes"
// 	"encoding/json"
// 	// "fmt"
// 	"io/ioutil"
// 	"net/http"
// 	"os"

// 	"github.com/gin-gonic/gin"
// )

// // DataPurchaseRequest represents the request payload for purchasing data
// type Data1PurchaseRequest struct {
// 	Destination string `json:"destination"`
// 	Plan        string `json:"plan"`
// }

// // // Response represents the generic response structure
// type Data1Response struct {
// 	Status  string      `json:"status"`
// 	Message string      `json:"message"`
// 	Result  interface{} `json:"result,omitempty"`
// }

// // DataPurchaseHandler handles the data purchase process
// func DataPurchaseHandler(c *gin.Context) {
// 	var dataPurchaseRequest DataPurchaseRequest
// 	if err := c.ShouldBindJSON(&dataPurchaseRequest); err != nil {
// 		c.JSON(http.StatusBadRequest, Response{
// 			Status:  "error",
// 			Message: "Invalid JSON input: " + err.Error(),
// 		})
// 		return
// 	}

// 	// Send request to Dojah API
// 	dojahURL := "https://api.dojah.io/api/v1/purchase/data"
// 	dojahData, err := json.Marshal(dataPurchaseRequest)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, Response{
// 			Status:  "error",
// 			Message: "Failed to marshal JSON: " + err.Error(),
// 		})
// 		return
// 	}

// 	req, err := http.NewRequest("POST", dojahURL, bytes.NewBuffer(dojahData))
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, Response{
// 			Status:  "error",
// 			Message: "Failed to create request: " + err.Error(),
// 		})
// 		return
// 	}
// 	req.Header.Set("AppId", os.Getenv("DOJAH_APP_ID"))
// 	req.Header.Set("Authorization", os.Getenv("DOJAH_SECRET_KEY"))
// 	req.Header.Set("Content-Type", "application/json")

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

// 	if resp.StatusCode != http.StatusOK {
// 		c.JSON(resp.StatusCode, Response{
// 			Status:  "error",
// 			Message: "Failed to purchase data",
// 			Result:  json.RawMessage(body),
// 		})
// 		return
// 	}

// 	var dojahResponse map[string]interface{}
// 	if err := json.Unmarshal(body, &dojahResponse); err != nil {
// 		c.JSON(http.StatusInternalServerError, Response{
// 			Status:  "error",
// 			Message: "Failed to parse response: " + err.Error(),
// 			Result:  json.RawMessage(body),
// 		})
// 		return
// 	}

// 	c.JSON(http.StatusOK, Response{
// 		Status:  "success",
// 		Message: "Data purchase successful",
// 		Result:  dojahResponse,
// 	})
// }
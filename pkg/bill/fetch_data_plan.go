package bill

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// DataPlansResponse represents the response payload for available data plans
// type DataPlansResponse struct {
// 	Entity []struct {
// 		Amount  int64 `json:"amount"`
// 		Plan      string `json:"plan"`
// 		Description     string `json:"description"`
// 	} `json:"entity"`
// }

// DataPlansHandler handles the request to view available data plans
func DataPlansHandler(c *gin.Context) {
	// Prepare the request
	url := "https://api.dojah.io/api/v1/purchase/data/plans"
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
	req.Header.Set("accept", "text/plain")

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
			Message: "Failed to retrieve data plans",
			Result:  string(body),
		})
		return
	}

	var dataPlansResponse DataPlansResponse
	if err := json.Unmarshal(body, &dataPlansResponse); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to parse response: " + err.Error(),
			Result:  json.RawMessage(body),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Status:  "success",
		Message: "Data plans retrieved successfully",
		Result:  dataPlansResponse.Entity,
	})
}
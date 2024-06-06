package kyc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	// "log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	// "github.com/joho/godotenv"
	"go_code/database"
)

// PhotoIDVerification represents the request payload for photo ID verification
type PhotoIDVerification struct {
	PhotoIDImage string `json:"photoid_image"`
	SelfieImage  string `json:"selfie_image"`
	UserID       int    `json:"user_id"`
}

// PhotoIDVerificationResponse represents the response payload for photo ID verification
type PhotoIDVerificationResponse struct {
	Entity struct {
		Selfie struct {
			ConfidenceValue  float64 `json:"confidence_value"`
			Match            bool    `json:"match"`
			PhotoIDImageBlurry bool    `json:"photoId_image_blurry"`
			SelfieImageBlurry bool    `json:"selfie_image_blurry"`
			SelfieGlare      bool    `json:"selfie_glare"`
			PhotoIDGlare     bool    `json:"photoId_glare"`
			AgeRange         string  `json:"age_range"`
			Sunglasses       bool    `json:"sunglasses"`
			CardType         string  `json:"card_type"`
		} `json:"selfie"`
	} `json:"entity"`
	Error string `json:"error"`
}

// Response represents the generic response structure
type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result,omitempty"`
}

// PhotoIDVerificationHandler handles the photo ID verification process
func PhotoIDVerificationHandler(c *gin.Context) {
	// Load environment variables
	// if err := godotenv.Load(); err != nil {
	// 	log.Fatalf("Error loading .env file")
	// }

	var verificationRequest PhotoIDVerification
	if err := c.ShouldBindJSON(&verificationRequest); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Status:  "error",
			Message: "Invalid JSON input: " + err.Error(),
		})
		return
	}

	// Create the request payload
	payload := map[string]string{
		"photoid_image": verificationRequest.PhotoIDImage,
		"selfie_image":  verificationRequest.SelfieImage,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to marshal JSON: " + err.Error(),
		})
		return
	}

	// Prepare the request
	url := "https://sandbox.dojah.io/api/v1/kyc/photoid/verify"
	// url := "https://sandbox.dojah.io/api/v1/kyc/photoid/verify"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadJSON))
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to create request: " + err.Error(),
		})
		return
	}

	appID := os.Getenv("DOJAH_APP_ID")
	secretKey := os.Getenv("DOJAH_SECRET_KEY")

	// Debugging logs
	// log.Printf("App ID: %s", appID)
	// log.Printf("Secret Key: %s", secretKey)

	req.Header.Set("AppId", appID)
	req.Header.Set("Authorization", secretKey)
	req.Header.Set("accept", "text/plain")
	req.Header.Set("content-type", "application/json")

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

	var verificationResponse PhotoIDVerificationResponse
	if err := json.Unmarshal(body, &verificationResponse); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Status:  "error",
			Message: "Failed to parse response: " + err.Error(),
			Result:  json.RawMessage(body),
		})
		return
	}

	// Check for errors in the API response
	if verificationResponse.Error != "" {
		c.JSON(http.StatusBadRequest, Response{
			Status:  "error",
			Message: verificationResponse.Error,
		})
		return
	}

	// Check the match status
	if verificationResponse.Entity.Selfie.Match {
		// Update the users table to set biometric_kyc to true
		if err := updateUserBiometricKYC(verificationRequest.UserID); err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Status:  "error",
				Message: "Failed to update user data: " + err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, Response{
			Status:  "success",
			Message: "Liveness verification successful",
			Result:  verificationResponse.Entity,
		})
	} else {
		c.JSON(http.StatusOK, Response{
			Status:  "error",
			Message: "Liveness verification failed",
			Result:  verificationResponse.Entity,
		})
	}
}

// updateUserBiometricKYC updates the user's biometric KYC status in the database
func updateUserBiometricKYC(userID int) error {
	db, err := database.PostgreSQLConnect()
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close(context.Background())

	_, err = db.Exec(context.Background(),
		"UPDATE users SET biometric_kyc = true WHERE user_id = $1", userID)
	if err != nil {
		return fmt.Errorf("failed to update user data: %w", err)
	}
	return nil
}
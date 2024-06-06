package main

import (
	"go_code/pkg/auth"
	"go_code/pkg/user"
	"go_code/pkg/wallet"
	"go_code/pkg/transaction"
	"go_code/pkg/kyc"
	"go_code/pkg/bill"
	"go_code/pkg/webhook"
	"go_code/pkg/third_party"
	"github.com/joho/godotenv"
	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v4/stdlib" // Import the PostgreSQL driver
)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		return
	}

	// Initialize the Gin router
	application := gin.Default()

	// Define API endpoints and their handlers
	//User API
	application.POST("/user", user.UserRegistration)
	application.GET("/user/:user_id", user.FetchSingleUser)
	
	//Auth API
	application.POST("/auth/login", auth.UserLogin)
	application.POST("/auth/reset_password", auth.UserResetPassword)
	application.POST("/auth/forgot_password", auth.ForgotPassword)
	application.POST("/auth/verify_pin", auth.VerifyPin)

	//Wallet API
	// Create wallet
	application.POST("/wallet", wallet.CreateCustomerHandler)

	// View single user wallet
	application.GET("/wallet/:user_id", wallet.ViewWalletHandler)

	// View all banks
	application.GET("/wallet/banks", wallet.ViewAllBanksHandler)

	// Check Dojah balance
	application.GET("/dojah_balance", third_party_balance.BalanceHandler)

	// Transactions API
	application.POST("/transaction/transfer", transaction.FundTransferHandler)
	
	// KYC
	application.POST("/biometric_kyc", kyc.PhotoIDVerificationHandler)

	// Bill
	// Airtime purchase
	application.POST("/bill/airtime_purchase", bill.AirtimePurchaseHandler)

	// Data purchase
	application.POST("/bill/data_purchase", bill.DataPurchaseHandler)
	
	// Fetch all data plans
	application.GET("/bill/data_plan", bill.DataPlansHandler)

	// Webhook
	application.POST("/webhook", webhook.WebhookHandler)


    // Run the application on port 8081
    application.Run(":8081")



	application.GET("/customer/:emailOrCode", transaction.GetCustomerHandler)
	application.GET("/dedicated_account/:dedicatedAccountId", transaction.GetDedicatedAccountHandler)

	// Run the application on port 8081
	application.Run(":8081")
}
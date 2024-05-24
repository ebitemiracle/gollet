package main

import (
	"go_code/pkg/auth"
	"go_code/pkg/user"
	"go_code/pkg/wallet"
	"go_code/pkg/transaction"
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
	application.POST("/wallet", wallet.CreateCustomerHandler)
	application.GET("/wallet/:user_id", wallet.ViewWalletHandler)

	// Bank API
	application.GET("/wallet/banks", wallet.ViewAllBanksHandler)

	// Transactions API
	application.POST("/transaction/transfer", transaction.FundTransferHandler)
	
	application.GET("/customer/:emailOrCode", transaction.GetCustomerHandler)
	application.GET("/dedicated_account/:dedicatedAccountId", transaction.GetDedicatedAccountHandler)

	// Run the application on port 8081
	application.Run(":8081")
}


// package main

// import (
// 	"github.com/gin-gonic/gin"
// 	"go_code/pkg/user"
// 	"go_code/pkg/auth"
// 	_ "github.com/go-sql-driver/mysql"
// )

// func main() {

// 	// APIs
// 	application := gin.Default()

// 	application.POST("/user", user.UserRegistration)
// 	application.GET("/user/:user_id", user.FetchSingleUser)
	
// 	application.POST("/auth/login", auth.UserLogin)
// 	application.POST("/auth/reset_password", auth.UserResetPassword)
// 	application.POST("/auth/forgot_password", auth.ForgotPassword)
// 	application.POST("/auth/verify_pin", auth.VerifyPin)
// 	application.Run(":8081")
// }
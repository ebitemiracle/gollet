package main

import (
	"github.com/gin-gonic/gin"
	"go_code/pkg/user"
	"go_code/pkg/auth"
	_ "github.com/jackc/pgx/v4/stdlib" // Import the PostgreSQL driver
)

func main() {
	// Initialize the Gin router
	application := gin.Default()

	// Define API endpoints and their handlers
	application.POST("/user", user.UserRegistration)
	application.GET("/user/:user_id", user.FetchSingleUser)
	
	application.POST("/auth/login", auth.UserLogin)
	application.POST("/auth/reset_password", auth.UserResetPassword)
	application.POST("/auth/forgot_password", auth.ForgotPassword)
	application.POST("/auth/verify_pin", auth.VerifyPin)

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
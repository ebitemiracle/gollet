package wallet

import (
	"context"
	"go_code/database"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// View wallet information
// Wallet structs
type User struct {
    UserID    int64  `json:"user_id"`
    Email     string `json:"email"`
    FullName string `json:"fullname"`
    Phone     string `json:"phone"`
    CurrentBalance     float64 `json:"current_balance"`
}

type Wallet struct {
    WalletID     int64  `json:"wallet_id"`
    CustomerCode string  `json:"customer_code"`
    BankName     string `json:"bank_name"`
    BankID       int    `json:"bank_id"`
    BankSlug     string `json:"bank_slug"`
    AccountName  string `json:"account_name"`
    AccountNumber string `json:"account_number"`
    DVAid int64 `json:"dva_id"`
    CurrentBalance int64 `json:"current_balance"`
}

type UserWalletResponse struct {
    User    User     `json:"user"`
    Wallets []Wallet `json:"wallets"`
}

// ViewWalletHandler handles the request to view user wallet information
func ViewWalletHandler(c *gin.Context) {
    userIDParam := c.Param("user_id")
    userID, err := strconv.ParseInt(userIDParam, 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, Response{
            Status:  "error",
            Message: "Invalid user_id parameter",
        })
        return
    }

    userWalletResponse, err := fetchUserAndWallets(userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, Response{
            Status:  "error",
            Message: "Failed to fetch user and wallet information: " + err.Error(),
        })
        return
    }

    if userWalletResponse == nil {
        c.JSON(http.StatusNotFound, Response{
            Status:  "error",
            Message: "User not found or no wallets available",
        })
        return
    }

    c.JSON(http.StatusOK, Response{
        Status:  "success",
        Message: "User wallet information retrieved successfully",
        Result:  userWalletResponse,
    })
}


// fetchUserAndWallets fetches user and wallet information from the database
func fetchUserAndWallets(userID int64) (*UserWalletResponse, error) {
    db, err := database.PostgreSQLConnect()
    if err != nil {
        return nil, err
    }
    defer db.Close(context.Background())

    // Fetch user information
    var user User
    userQuery := "SELECT user_id, email, fullname, phone FROM users WHERE user_id = $1 AND deleted = false"
    err = db.QueryRow(context.Background(), userQuery, userID).Scan(&user.UserID, &user.Email, &user.FullName, &user.Phone)
    if err != nil {
        return nil, err
    }

    // Fetch wallet information
    walletQuery := "SELECT wallet_id, customer_code, current_balance, bank_name, bank_id, bank_slug, account_name, account_number, dva_id FROM wallet WHERE user_id = $1"
    rows, err := db.Query(context.Background(), walletQuery, userID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var wallets []Wallet
    for rows.Next() {
        var wallet Wallet
        err := rows.Scan(&wallet.WalletID, &wallet.CustomerCode, &wallet.CurrentBalance, &wallet.BankName, &wallet.BankID, &wallet.BankSlug, &wallet.AccountName, &wallet.AccountNumber, &wallet.DVAid)
        if err != nil {
            return nil, err
        }
        wallets = append(wallets, wallet)
    }

    return &UserWalletResponse{
        User:    user,
        Wallets: wallets,
    }, nil
}
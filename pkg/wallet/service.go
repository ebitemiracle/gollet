package wallet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"strconv"

	"github.com/gin-gonic/gin"
	"go_code/database"
)

// Customer represents the customer data structure for Paystack API
type Customer struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
}

// Response represents the API response structure
type Response struct {
	Status     string      `json:"status"`
	StatusCode int         `json:"status_code,omitempty"`
	Message    string      `json:"message"`
	Result     interface{} `json:"result,omitempty"`
}

// CreateCustomerRequest represents the request body for creating a Paystack customer
type CreateCustomerRequest struct {
	UserID int64 `json:"user_id"`
}

type CreateCustomerResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Email        string `json:"email"`
		CustomerCode string `json:"customer_code"`
	} `json:"data"`
}

type CreateDVAResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Bank struct {
			Name string `json:"name"`
			ID   int    `json:"id"`
			Slug string `json:"slug"`
		} `json:"bank"`
		Customer struct{
			CustomerCode string `json:"customer_code"`
		} `json:"customer"`
		AccountName   string `json:"account_name"`
		AccountNumber string `json:"account_number"`
		DVAid int64 `json:"id"`
	} `json:"data"`
}


// CreateCustomerHandler handles the customer creation and DVA process
func CreateCustomerHandler(c *gin.Context) {
	var response Response
	var request CreateCustomerRequest

	// Bind JSON data from the request body to the CreateCustomerRequest struct
	if err := c.ShouldBindJSON(&request); err != nil {
		response = Response{
			Status:     "error",
			StatusCode: http.StatusBadRequest,
			Message:    "Malformed JSON request: " + err.Error(),
		}
		c.JSON(response.StatusCode, response)
		return
	}

	// Fetch user information from the database
	user, err := fetchUserFromDatabase(request.UserID)
	if err != nil {
		response = Response{
			Status:     "error",
			StatusCode: http.StatusInternalServerError,
			Message:    "Failed to fetch user information: " + err.Error(),
		}
		c.JSON(response.StatusCode, response)
		return
	}

	if user == nil {
		response = Response{
			Status:     "error",
			StatusCode: http.StatusNotFound,
			Message:    "User not found",
		}
		c.JSON(response.StatusCode, response)
		return
	}

	// Create a customer with Paystack using the fetched user information
	customer := Customer{
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Phone:     user.Phone,
	}

	customerCode, err := createCustomerWithPaystack(customer)
	if err != nil {
		response = Response{
			Status:     "error",
			StatusCode: http.StatusInternalServerError,
			Message:    "Failed to create customer: " + err.Error(),
		}
		c.JSON(response.StatusCode, response)
		return
	}

	// Create a DVA with Paystack using the customer code
	dvaData, err := createDVAWithPaystack(customerCode)
	if err != nil {
		response = Response{
			Status:     "error",
			StatusCode: http.StatusInternalServerError,
			Message:    "Failed to create DVA: " + err.Error(),
		}
		c.JSON(response.StatusCode, response)
		return
	}

	// Save DVA information in the database
	if err := saveDVAInDatabase(request.UserID, dvaData); err != nil {
		response = Response{
			Status:     "error",
			StatusCode: http.StatusInternalServerError,
			Message:    "Oops! " + err.Error(),
		}
		c.JSON(response.StatusCode, response)
		return
	}

	response = Response{
		Status:     "success",
		StatusCode: http.StatusOK,
		Message:    "Wallet created successfully",
		Result:     dvaData,
	}
	c.JSON(response.StatusCode, response)
}

// fetchUserFromDatabase fetches user information from the database based on user_id
func fetchUserFromDatabase(userID int64) (*Customer, error) {
	db, err := database.PostgreSQLConnect()
	if err != nil {
		return nil, err
	}
	defer db.Close(context.Background())

	var user Customer
	query := "SELECT email, fullname, phone FROM users WHERE user_id = $1 AND deleted = false"
	err = db.QueryRow(context.Background(), query, userID).Scan(&user.Email, &user.FirstName, &user.Phone)
	if err != nil {
		return nil, err
	}

	// Split the fullname into first and last name (assuming full name is stored in the "fullname" field)
	names := splitFullName(user.FirstName)
	user.LastName = names[0]
	user.FirstName = names[1]

	return &user, nil
}

// splitFullName splits a full name into first name and last name
func splitFullName(fullname string) [2]string {
	var names [2]string
	parts := strings.SplitN(fullname, " ", 2)
	names[0] = parts[0]
	if len(parts) > 1 {
		names[1] = parts[1]
	}
	return names
}

// createCustomerWithPaystack sends a request to Paystack to create a customer
func createCustomerWithPaystack(customer Customer) (string, error) {
	url := "https://api.paystack.co/customer"
	authorization := "Bearer " + os.Getenv("PAYSTACK_SECRET_KEY")
	contentType := "application/json"

	data, err := json.Marshal(customer)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", authorization)
	req.Header.Set("Content-Type", contentType)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Error creating customer: %s", body)
	}

	var result CreateCustomerResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if !result.Status {
		return "", fmt.Errorf("Error creating customer: %s", result.Message)
	}

	return result.Data.CustomerCode, nil
}

// createDVAWithPaystack sends a request to Paystack to create a dedicated virtual account
func createDVAWithPaystack(customerCode string) (*CreateDVAResponse, error) {
	url := "https://api.paystack.co/dedicated_account"
	authorization := "Bearer " + os.Getenv("PAYSTACK_SECRET_KEY")
	contentType := "application/json"

	data := map[string]interface{}{
		"customer":      customerCode,
		"preferred_bank": "wema-bank",
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", authorization)
	req.Header.Set("Content-Type", contentType)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Error creating wallet: %s", body)
	}

	var result CreateDVAResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if !result.Status {
		return nil, fmt.Errorf("Error creating wallet: %s", result.Message)
	}

	return &result, nil
}

// checkDVAExists checks if a DVA already exists in the database
func checkDVAExists(userID int64, dvaData *CreateDVAResponse) (bool, error) {
	db, err := database.PostgreSQLConnect()
	if err != nil {
		return false, err
	}
	defer db.Close(context.Background())

	query := `
		SELECT COUNT(*) 
		FROM wallet 
		WHERE user_id = $1 AND customer_code = $2 AND deleted = false
	`
	var count int
	err = db.QueryRow(context.Background(), query, userID,dvaData.Data.Customer.CustomerCode).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}


// saveDVAInDatabase saves the DVA information in the database
func saveDVAInDatabase(userID int64, dvaData *CreateDVAResponse) error {

	exists, err := checkDVAExists(userID, dvaData)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("wallet address has already been created for you")
	}
	

	db, err := database.PostgreSQLConnect()
	if err != nil {
		return err
	}
	defer db.Close(context.Background())

	query := `
		INSERT INTO wallet (user_id, customer_code, bank_name, bank_id, bank_slug, account_name, account_number, dva_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = db.Exec(context.Background(), query, userID, dvaData.Data.Customer.CustomerCode, dvaData.Data.Bank.Name, dvaData.Data.Bank.ID, dvaData.Data.Bank.Slug, dvaData.Data.AccountName, dvaData.Data.AccountNumber, dvaData.Data.DVAid)
	if err != nil {
		return err
	}

	return nil
}



// View wallet information
// Wallet structs
type User struct {
    UserID    int64  `json:"user_id"`
    Email     string `json:"email"`
    FullName string `json:"fullname"`
    Phone     string `json:"phone"`
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
    walletQuery := "SELECT wallet_id, customer_code, bank_name, bank_id, bank_slug, account_name, account_number, dva_id FROM wallet WHERE user_id = $1"
    rows, err := db.Query(context.Background(), walletQuery, userID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var wallets []Wallet
    for rows.Next() {
        var wallet Wallet
        err := rows.Scan(&wallet.WalletID, &wallet.CustomerCode, &wallet.BankName, &wallet.BankID, &wallet.BankSlug, &wallet.AccountName, &wallet.AccountNumber, &wallet.DVAid)
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



// View all banks
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

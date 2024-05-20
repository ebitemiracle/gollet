package user

import (
	"fmt"
	"go_code/database"
	"net/http"
	"regexp"
	// "sync"

	"golang.org/x/crypto/bcrypt"

	"github.com/gin-gonic/gin"
)

type User struct {
	ID         int64  `json:"user_id"`
	Fullname   string `json:"fullname"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	Password   string `json:"password"`
	Password_2 string `json:"password_2"`
	Deleted    string `json:"deleted"`
}

type Response struct {
	Status     string      `json:"status"`
	StatusCode int         `json:"status_code,omitempty"`
	Message    string      `json:"message"`
	Result     interface{} `json:"result,omitempty"`
}

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	phoneRegex = regexp.MustCompile(`^\+?[0-9\s\-\(\)\.]{10,15}$`)
)

func IsValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

func IsValidPhone(phone string) bool {
	return phoneRegex.MatchString(phone)
}

func getHashedPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func handleDatabaseError(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, Response{
		Status:  "error",
		Message: "Database error: " + err.Error(),
	})
}

func UserRegistration(c *gin.Context) {
	var response Response
	var newUser User

	if err := c.ShouldBindJSON(&newUser); err != nil {
		response = Response{
			Status:     "error",
			StatusCode: http.StatusBadRequest,
			Message:    "Malformed JSON request: " + err.Error(),
		}
		c.JSON(response.StatusCode, response)
		return
	}

	if err := validateUserInput(newUser); err != nil {
		response = Response{
			Status:     "error",
			StatusCode: http.StatusBadRequest,
			Message:    err.Error(),
		}
		c.JSON(response.StatusCode, response)
		return
	}

	if err := validateUserPassword(newUser); err != nil {
		response = Response{
			Status:     "error",
			StatusCode: http.StatusBadRequest,
			Message:    err.Error(),
		}
		c.JSON(response.StatusCode, response)
		return
	}

	hashedPassword, err := getHashedPassword(newUser.Password)
	if err != nil {
		handleDatabaseError(c, err)
		return
	}

	db, err := database.MySQLConnect()
	if err != nil {
		handleDatabaseError(c, err)
		return
	}
	defer db.Close()

	result, err := db.Exec("INSERT INTO gollet (fullname, email, phone, password) VALUES (?, ?, ?, ?)",
		newUser.Fullname, newUser.Email, newUser.Phone, hashedPassword)
	if err != nil {
		handleDatabaseError(c, err)
		return
	}

	userID, err := result.LastInsertId()
	if err != nil {
		handleDatabaseError(c, err)
		return
	}

	newUser.ID = userID
	response = Response{
		Status:     "success",
		StatusCode: http.StatusCreated,
		Message:    "Account created successfully",
		Result: struct {
			ID       int64  `json:"user_id"`
			Fullname string `json:"fullname"`
			Email    string `json:"email"`
			Phone    string `json:"phone"`
		}{
			ID:       newUser.ID,
			Fullname: newUser.Fullname,
			Email:    newUser.Email,
			Phone:    newUser.Phone,
		},
	}
	c.JSON(response.StatusCode, response)
}

func FetchSingleUser(c *gin.Context) {
	var response Response

	db, err := database.MySQLConnect()
	if err != nil {
		response.Status = "error"
		response.StatusCode = http.StatusInternalServerError
		response.Message = "Failed to connect to the database. Reason: " + err.Error()
		c.JSON(response.StatusCode, response)
		return
	}
	defer db.Close()

	userID := c.Param("user_id")

	var newUser User
	err = db.QueryRow("SELECT user_id, fullname, email, phone, deleted FROM gollet WHERE user_id = ? AND deleted=0 LIMIT 1", userID).
		Scan(&newUser.ID, &newUser.Fullname, &newUser.Email, &newUser.Phone, &newUser.Deleted)
	if err != nil {
		response.Status = "error"
		response.StatusCode = http.StatusInternalServerError
		response.Message = "error: " + err.Error()
	} else {
		response.Status = "success"
		response.StatusCode = http.StatusOK
		response.Message = "Record fetched successfully"
		response.Result = struct {
			ID       int64  `json:"user_id"`
			Fullname string `json:"fullname"`
			Email    string `json:"email"`
			Phone    string `json:"phone"`
			Deleted  string `json:"deleted"`
		}{
			ID:       newUser.ID,
			Fullname: newUser.Fullname,
			Email:    newUser.Email,
			Phone:    newUser.Phone,
			Deleted:  newUser.Deleted,
		}
	}

	c.JSON(response.StatusCode, response)
}

func DoesUserExist(userEmail string) (int, error) {
	db, err := database.MySQLConnect()
	if err != nil {
		return 0, fmt.Errorf("failed to connect to the database: %w", err)
	}
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM gollet WHERE email = ? AND deleted=0", userEmail).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count rows: %w", err)
	}

	return count, nil
}

func DoesUserIdExist(userID int64) (int, error) {
	db, err := database.MySQLConnect()
	if err != nil {
		return 0, fmt.Errorf("failed to connect to the database: %w", err)
	}
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM gollet WHERE user_id = ? AND deleted=0", userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count rows: %w", err)
	}

	return count, nil
}

func validateUserInput(user User) error {
	if user.Fullname == "" {
		return fmt.Errorf("full name is required")
	}
	if user.Email == "" {
		return fmt.Errorf("email is required")
	}
	if user.Phone == "" {
		return fmt.Errorf("phone number is required")
	}
	if !IsValidEmail(user.Email) {
		return fmt.Errorf("invalid email address")
	}
	if !IsValidPhone(user.Phone) {
		return fmt.Errorf("invalid phone number")
	}

	countEmail, err := DoesUserExist(user.Email)
	if err != nil {
		return fmt.Errorf("error encountered")
	}
	if countEmail > 0 {
		return fmt.Errorf("user with this email address already exists. Try a different email address")
	}
	return nil
}

func validateUserPassword(user User) error {
	if user.Password == "" {
		return fmt.Errorf("password is required")
	}
	if user.Password_2 == "" {
		return fmt.Errorf("confirm password is required")
	}
	if len(user.Password) < 4 {
		return fmt.Errorf("password must be at least 4 characters")
	}
	if user.Password != user.Password_2 {
		return fmt.Errorf("passwords do not match")
	}
	return nil
}
package auth

import (
    "crypto/rand"
    "fmt"
    "net/http"
    "net/smtp"
    "time"

    "github.com/gin-gonic/gin"
    "golang.org/x/crypto/bcrypt"
    "go_code/database"
)

// User represents the user data structure
type User struct {
    ID         int64  `json:"user_id"`
    Fullname   string `json:"fullname"`
    Email      string `json:"email"`
    Phone      string `json:"phone"`
    Password   string `json:"password"`
    Password_2 string `json:"password_2"`
    Deleted    string `json:"deleted"`
}

// ForgotPasswordRequest represents the request body for forgot password
type ForgotPasswordRequest struct {
    Email  string `json:"email"`
    UserID int64  `json:"user_id"`
}

// VerifyPinRequest represents the request body for verifying the PIN
type VerifyPinRequest struct {
    Email           string `json:"email"`
    UserID          int64  `json:"user_id"`
    Pin             string `json:"pin"`
    NewPassword     string `json:"new_password"`
    ConfirmPassword string `json:"confirm_password"`
}

// Response represents the API response structure
type Response struct {
    Status     string      `json:"status"`
    StatusCode int         `json:"status_code,omitempty"`
    Message    string      `json:"message"`
    Result     interface{} `json:"result,omitempty"`
}

// GenerateRandomPIN generates a random 4-digit PIN
func generateRandomPIN() string {
    b := make([]byte, 2)
    _, err := rand.Read(b)
    if err != nil {
        return "0000"
    }
    return fmt.Sprintf("%04x", b[:2])
}

// SendEmail sends an email with the specified subject and bodyg
func sendEmail(to string, subject string, body string) error {
    from := "nonso@solaa.app"
    password := "nonso@solaa.app"
    smtpHost := "solaa.app"
    smtpPort := "587"

    msg := "From: " + from + "\n" +
        "To: " + to + "\n" +
        "Subject: " + subject + "\n\n" +
        body

    auth := smtp.PlainAuth("", from, password, smtpHost)
    return smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{to}, []byte(msg))
}

// GetHashedPassword hashes a password using bcrypt
func getHashedPassword(password string) (string, error) {
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return "", err
    }
    return string(hash), nil
}

// HandleDatabaseError handles database errors and sends a JSON response
func handleDatabaseError(c *gin.Context, err error) {
    c.JSON(http.StatusInternalServerError, Response{
        Status:  "error",
        Message: "Database error: " + err.Error(),
    })
}

// UserResetPasswordRequest represents the request body for resetting password
type UserResetPasswordRequest struct {
    ID               int64  `json:"user_id"`
    Email            string `json:"email"`
    PreviousPassword string `json:"previous_password"`
    NewPassword      string `json:"new_password"`
    ConfirmPassword  string `json:"confirm_password"`
}

// ValidateNewPassword validates the new password and confirmation
func validateNewPassword(newPassword, confirmPassword string) error {
    if newPassword == "" {
        return fmt.Errorf("new password is required")
    }
    if confirmPassword == "" {
        return fmt.Errorf("confirm password is required")
    }
    if newPassword != confirmPassword {
        return fmt.Errorf("passwords do not match")
    }
    if len(newPassword) < 4 {
        return fmt.Errorf("password must be at least 4 characters")
    }
    return nil
}

// UserLogin handles the user login process
func UserLogin(c *gin.Context) {
    var response Response

    // Connect to the database
    db, err := database.MySQLConnect()
    if err != nil {
        response.Status = "error"
        response.StatusCode = http.StatusInternalServerError
        response.Message = "Failed to connect to the database. Reason: " + err.Error()
        c.JSON(response.StatusCode, response)
        return
    }
    defer db.Close()

    // Bind JSON data from the request body to a User struct
    var newUser User
    if err := c.ShouldBindJSON(&newUser); err != nil {
        response.Status = "error"
        response.StatusCode = http.StatusBadRequest
        response.Message = "Malformed JSON request. Reason: " + err.Error()
        c.JSON(response.StatusCode, response)
        return
    }

    // Query the database to fetch the user details
    var storedUser User
    var storedPassword string
    row := db.QueryRow("SELECT user_id, fullname, email, phone, password, deleted FROM gollet WHERE email = ? AND deleted = 0 LIMIT 1", newUser.Email)
    err = row.Scan(&storedUser.ID, &storedUser.Fullname, &storedUser.Email, &storedUser.Phone, &storedPassword, &storedUser.Deleted)
    if err != nil {
        response.Status = "error"
        response.StatusCode = http.StatusUnauthorized
        response.Message = "Login failed: User not found or incorrect email"
        c.JSON(response.StatusCode, response)
        return
    }

    // Compare the hashed password with the provided password
    err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(newUser.Password))
    if err != nil {
        response.Status = "error"
        response.StatusCode = http.StatusUnauthorized
        response.Message = "Login failed: Incorrect password"
        c.JSON(response.StatusCode, response)
        return
    }

    // Successful login
    response.Status = "success"
    response.StatusCode = http.StatusOK
    response.Message = "Login successful"
    response.Result = struct {
        ID       int64  `json:"user_id"`
        Fullname string `json:"fullname"`
        Email    string `json:"email"`
        Phone    string `json:"phone"`
        Deleted  string `json:"deleted"`
    }{
        ID:       storedUser.ID,
        Fullname: storedUser.Fullname,
        Email:    storedUser.Email,
        Phone:    storedUser.Phone,
        Deleted:  storedUser.Deleted,
    }

    // Return the response as JSON
    c.JSON(response.StatusCode, response)
}
// UserResetPassword handles the user reset password process
func UserResetPassword(c *gin.Context) {
    var response Response
    var request UserResetPasswordRequest

    // Bind JSON data from the request body
    if err := c.ShouldBindJSON(&request); err != nil {
        response.Status = "error"
        response.StatusCode = http.StatusBadRequest
        response.Message = "Malformed JSON request. Reason: " + err.Error()
        c.JSON(response.StatusCode, response)
        return
    }

    // Connect to the database
    db, err := database.MySQLConnect()
    if err != nil {
        handleDatabaseError(c, err)
        return
    }
    defer db.Close()

    // Validate new password
    if err := validateNewPassword(request.NewPassword, request.ConfirmPassword); err != nil {
        response.Status = "error"
        response.StatusCode = http.StatusBadRequest
        response.Message = err.Error()
        c.JSON(response.StatusCode, response)
        return
    }

    // Query the database to verify the previous password
    var storedPassword string
    err = db.QueryRow("SELECT password FROM gollet WHERE email = ? AND user_id = ? AND deleted = 0", request.Email, request.ID).Scan(&storedPassword)
    if err != nil {
        response.Status = "error"
        response.StatusCode = http.StatusNotFound
        response.Message = "User not found or incorrect previous password. Reason: " + err.Error()
        c.JSON(response.StatusCode, response)
        return
    }

    // Check if the previous password is correct
    err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(request.PreviousPassword))
    if err != nil {
        response.Status = "error"
        response.StatusCode = http.StatusUnauthorized
        response.Message = "Incorrect previous password"
        c.JSON(response.StatusCode, response)
        return
    }

    // Hash the new password
    hashedNewPassword, err := getHashedPassword(request.NewPassword)
    if err != nil {
        response.Status = "error"
        response.StatusCode = http.StatusInternalServerError
        response.Message = "Failed to hash new password. Reason: " + err.Error()
        c.JSON(response.StatusCode, response)
        return
    }

    // Update the password in the database
    _, err = db.Exec("UPDATE gollet SET password = ? WHERE user_id = ?", hashedNewPassword, request.ID)
    if err != nil {
        response.Status = "error"
        response.StatusCode = http.StatusInternalServerError
        response.Message = "Failed to update password. Reason: " + err.Error()
        c.JSON(response.StatusCode, response)
        return
    }

    response.Status = "success"
    response.StatusCode = http.StatusOK
    response.Message = "Password updated successfully"
    c.JSON(response.StatusCode, response)
}

// ForgotPassword handles the forgot password process
func ForgotPassword(c *gin.Context) {
    var response Response
    var request ForgotPasswordRequest

    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid request. Reason: " + err.Error()})
        return
    }

    db, err := database.MySQLConnect()
    if err != nil {
        handleDatabaseError(c, err)
        return
    }
    defer db.Close()

    var user User
    if request.Email != "" {
        err = db.QueryRow("SELECT user_id, email FROM gollet WHERE email = ? AND deleted = 0", request.Email).Scan(&user.ID, &user.Email)
    }

    if err != nil {
        response.Status = "error"
        response.StatusCode = http.StatusUnauthorized
        response.Message = "User not found"
        c.JSON(response.StatusCode, response)
        return
    }

    pin := generateRandomPIN()
    _, err = db.Exec("UPDATE gollet SET reset_pin = ?, reset_pin_expiry = ?, pin_used = FALSE WHERE user_id = ?", pin, time.Now().Add(15*time.Minute), user.ID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Failed to save PIN"})
        return
    }

    err = sendEmail(user.Email, "Password Reset PIN", "Your password reset PIN is: "+pin)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Failed to send email"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"status": "success", "message": "PIN sent to email"})
}

// VerifyPin handles the verification of the reset PIN and password update
func VerifyPin(c *gin.Context) {
    var request VerifyPinRequest
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid request"})
        return
    }

    if request.NewPassword != request.ConfirmPassword {
        c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Passwords do not match"})
        return
    }

    db, err := database.MySQLConnect()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Database connection failed. Reason: " + err.Error()})
        return
    }
    defer db.Close()

    var storedPin string
    var expiryStr string
    var pinUsed bool

    err = db.QueryRow("SELECT reset_pin, reset_pin_expiry, pin_used FROM gollet WHERE email = ? AND user_id = ?", request.Email, request.UserID).Scan(&storedPin, &expiryStr, &pinUsed)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "User not found. Reason: " + err.Error()})
        return
    }

    if pinUsed {
        c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "PIN has already been used"})
        return
    }

    expiry, err := time.Parse("2006-01-02 15:04:05", expiryStr)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Failed to parse expiry time. Reason: " + err.Error()})
        return
    }

    if storedPin != request.Pin || time.Now().After(expiry) {
        c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Invalid or expired PIN"})
        return
    }

    hashedPassword, err := getHashedPassword(request.NewPassword)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Failed to hash password. Reason: " + err.Error()})
        return
    }

    _, err = db.Exec("UPDATE gollet SET password = ?, reset_pin = NULL, reset_pin_expiry = NULL, pin_used = TRUE WHERE user_id = ?", hashedPassword, request.UserID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Failed to update password. Reason: " + err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Password updated successfully"})
}
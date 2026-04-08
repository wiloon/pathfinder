package user

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"pathfinder-api/email"
	"pathfinder-api/storage"
)

var sessionStore *sessions.CookieStore

func InitSession(store *sessions.CookieStore) {
	sessionStore = store
}

func hashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

func checkPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Register handles POST /api/auth/register
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
	Email    string `json:"email" binding:"required,email"`
}

func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request parameters"})
		return
	}

	var existing storage.User
	if err := storage.DB.Where("username = ? OR email = ?", req.Username, req.Email).First(&existing).Error; err == nil {
		if existing.Username == req.Username {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Username already taken"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Email already registered"})
		}
		return
	}

	hashed, err := hashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to process registration"})
		return
	}

	token, err := generateToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to process registration"})
		return
	}

	u := storage.User{
		Username:          req.Username,
		Email:             req.Email,
		Password:          hashed,
		Status:            "pending",
		VerificationToken: token,
		TokenExpiresAt:    time.Now().Add(48 * time.Hour),
	}
	if err := storage.DB.Create(&u).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to create user"})
		return
	}

	_ = email.SendVerificationEmail(req.Email, req.Username, token)

	session, err := sessionStore.Get(c.Request, "pathfinder-session")
	if err == nil {
		session.Values["user_id"] = u.ID
		_ = session.Save(c.Request, c.Writer)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "Registration successful. Please check your email to verify your account.",
		"status":   "pending",
		"user_id":  u.ID,
		"username": u.Username,
	})
}

// LoginRequest for POST /api/auth/login
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request parameters"})
		return
	}

	var u storage.User
	if err := storage.DB.Where("username = ? OR email = ?", req.Username, req.Username).First(&u).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Invalid username or password"})
		return
	}

	if !checkPassword(req.Password, u.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Invalid username or password"})
		return
	}

	session, err := sessionStore.Get(c.Request, "pathfinder-session")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Session error"})
		return
	}
	session.Values["user_id"] = u.ID
	if err := session.Save(c.Request, c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to save session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"user_id":  u.ID,
		"username": u.Username,
		"status":   u.Status,
	})
}

// Logout handles POST /api/auth/logout
func Logout(c *gin.Context) {
	session, err := sessionStore.Get(c.Request, "pathfinder-session")
	if err == nil {
		session.Options.MaxAge = -1
		_ = session.Save(c.Request, c.Writer)
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// VerifyEmail handles GET /api/auth/verify-email?token=...
func VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid activation link"})
		return
	}

	var u storage.User
	if err := storage.DB.Where("verification_token = ?", token).First(&u).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid or expired activation link"})
		return
	}

	if u.Status == "active" {
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Account already activated"})
		return
	}

	if time.Now().After(u.TokenExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Activation link has expired. Please request a new one."})
		return
	}

	if err := storage.DB.Model(&u).Updates(map[string]interface{}{
		"status":             "active",
		"verification_token": "",
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to activate account"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Email verified! You can now use all features."})
}

// ResendVerification handles POST /api/auth/resend-verification
type ResendRequest struct {
	Email string `json:"email" binding:"required,email"`
}

func ResendVerification(c *gin.Context) {
	var req ResendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": true})
		return
	}

	var u storage.User
	if err := storage.DB.Where("email = ?", req.Email).First(&u).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": true})
		return
	}

	token, err := generateToken()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": true})
		return
	}

	storage.DB.Model(&u).Updates(map[string]interface{}{
		"verification_token": token,
		"token_expires_at":   time.Now().Add(48 * time.Hour),
	})

	_ = email.SendVerificationEmail(req.Email, u.Username, token)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ForgotPassword handles POST /api/auth/forgot-password
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

func ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": true})
		return
	}

	var u storage.User
	if err := storage.DB.Where("email = ?", req.Email).First(&u).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"success": true})
		return
	}

	token, err := generateToken()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": true})
		return
	}

	storage.DB.Model(&u).Updates(map[string]interface{}{
		"reset_token":            token,
		"reset_token_expires_at": time.Now().Add(time.Hour),
	})

	_ = email.SendPasswordResetEmail(req.Email, u.Username, token)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ResetPassword handles POST /api/auth/reset-password
type ResetPasswordRequest struct {
	Token    string `json:"token" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}

func ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request parameters"})
		return
	}

	var u storage.User
	if err := storage.DB.Where("reset_token = ?", req.Token).First(&u).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid or expired reset link"})
		return
	}

	if time.Now().After(u.ResetTokenExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Reset link has expired. Please request a new one."})
		return
	}

	hashed, err := hashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to update password"})
		return
	}

	storage.DB.Model(&u).Updates(map[string]interface{}{
		"password":    hashed,
		"reset_token": "",
	})

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetMe handles GET /api/auth/me
func GetMe(c *gin.Context) {
	session, err := sessionStore.Get(c.Request, "pathfinder-session")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Not authenticated"})
		return
	}

	userIDVal, ok := session.Values["user_id"]
	if !ok || userIDVal == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Not authenticated"})
		return
	}

	var u storage.User
	if err := storage.DB.First(&u, userIDVal).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"user_id":  u.ID,
		"username": u.Username,
		"email":    u.Email,
		"status":   u.Status,
	})
}

// UpdateProfile handles POST /api/user/profile
func UpdateProfile(c *gin.Context) {
	userID := c.GetString("user_id")

	bio := c.PostForm("bio")

	var profile storage.UserProfile
	result := storage.DB.Where("user_id = ?", userID).First(&profile)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			profile = storage.UserProfile{UserID: userID}
		}
	}

	if bio != "" {
		profile.Bio = bio
	}

	fh, err := c.FormFile("resume")
	if err == nil {
		f, ferr := fh.Open()
		if ferr == nil {
			data, rerr := io.ReadAll(f)
			f.Close()
			if rerr == nil {
				profile.ResumeFilename = fh.Filename
				profile.ResumeMimeType = fh.Header.Get("Content-Type")
				if profile.ResumeMimeType == "" {
					profile.ResumeMimeType = "application/octet-stream"
				}
				profile.ResumeData = data
			}
		}
	}

	if profile.ID == 0 {
		storage.DB.Create(&profile)
	} else {
		storage.DB.Save(&profile)
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

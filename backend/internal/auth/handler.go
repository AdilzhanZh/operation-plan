package auth

import (
	"OperationPlan/internal/config"
	"OperationPlan/internal/middleware"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/mail"
	"net/smtp"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

var (
	uppercaseRegex = regexp.MustCompile(`[A-Z]`)
	lowercaseRegex = regexp.MustCompile(`[a-z]`)
	digitRegex     = regexp.MustCompile(`[0-9]`)
)

type Handler struct {
	db                     *sql.DB
	sessionTTLSeconds      int64
	bootstrapAdminUsername string
	bootstrapAdminPassword string
	smtpHost               string
	smtpPort               string
	smtpUsername           string
	smtpPassword           string
	smtpFromEmail          string
	smtpFromName           string
	otpTTLMinutes          int
	resetSessionTTLMinutes int
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type registerRequest struct {
	FirstName       string `json:"first_name" binding:"required"`
	LastName        string `json:"last_name" binding:"required"`
	MiddleName      string `json:"middle_name"`
	Position        string `json:"position" binding:"required"`
	Email           string `json:"email" binding:"required"`
	Username        string `json:"username" binding:"required"`
	Password        string `json:"password" binding:"required"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
}

type verifyRegisterCodeRequest struct {
	Email string `json:"email" binding:"required"`
	Code  string `json:"code" binding:"required"`
}

type requestPasswordResetCodeRequest struct {
	Email string `json:"email" binding:"required"`
}

type verifyPasswordResetCodeRequest struct {
	Email string `json:"email" binding:"required"`
	Code  string `json:"code" binding:"required"`
}

type completePasswordResetRequest struct {
	ResetToken      string `json:"reset_token" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
}

type changePasswordRequest struct {
	OldPassword     string `json:"old_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type loginUser struct {
	ID         int64  `json:"id"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	MiddleName string `json:"middle_name"`
	FullName   string `json:"full_name"`
	Username   string `json:"username"`
	Email      string `json:"email"`
	Position   string `json:"position"`
	Role       string `json:"role"`
}

type loginResponse struct {
	Token string    `json:"token"`
	User  loginUser `json:"user"`
}

type meResponse struct {
	User meUser `json:"user"`
}

type meUser struct {
	ID         int64  `json:"id"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	MiddleName string `json:"middle_name"`
	FullName   string `json:"full_name"`
	Username   string `json:"username"`
	Email      string `json:"email"`
	Position   string `json:"position"`
	Role       string `json:"role"`
}

type registrationCodePayload struct {
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	MiddleName   string `json:"middle_name"`
	Position     string `json:"position"`
	Email        string `json:"email"`
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
	Role         string `json:"role"`
}

type passwordResetCodePayload struct {
	UserID int64 `json:"user_id"`
}

var (
	errUserExists            = errors.New("username already exists")
	errEmailExists           = errors.New("email already exists")
	errPendingRequestExists  = errors.New("registration request already pending")
	errInvalidRole           = errors.New("invalid role")
	errPasswordValidation    = errors.New("password validation error")
	errInvalidVerification   = errors.New("invalid or expired verification code")
	errEmailNotConfigured    = errors.New("email delivery is not configured")
	errPasswordResetNotFound = errors.New("account with this email was not found")
)

func RegisterRoutes(router gin.IRoutes, db *sql.DB, cfg *config.Config) {
	sessionTTLHours := cfg.SessionTTLHours
	if sessionTTLHours <= 0 {
		sessionTTLHours = 24
	}

	otpTTLMinutes := cfg.OTPTTLMinutes
	if otpTTLMinutes <= 0 {
		otpTTLMinutes = 10
	}

	resetTTLMinutes := cfg.ResetSessionTTLMinutes
	if resetTTLMinutes <= 0 {
		resetTTLMinutes = 15
	}

	h := &Handler{
		db:                     db,
		sessionTTLSeconds:      int64((time.Duration(sessionTTLHours) * time.Hour) / time.Second),
		bootstrapAdminUsername: normalizeBootstrapAdminUsername(cfg.BootstrapAdminUsername),
		bootstrapAdminPassword: strings.TrimSpace(cfg.BootstrapAdminPassword),
		smtpHost:               strings.TrimSpace(cfg.SMTPHost),
		smtpPort:               strings.TrimSpace(cfg.SMTPPort),
		smtpUsername:           strings.TrimSpace(cfg.SMTPUsername),
		smtpPassword:           cfg.SMTPPassword,
		smtpFromEmail:          strings.TrimSpace(cfg.SMTPFromEmail),
		smtpFromName:           strings.TrimSpace(cfg.SMTPFromName),
		otpTTLMinutes:          otpTTLMinutes,
		resetSessionTTLMinutes: resetTTLMinutes,
	}

	if err := h.ensureUsersTable(); err != nil {
		panic(err)
	}
	if err := h.ensureBootstrapAdmin(); err != nil {
		panic(err)
	}
	if err := h.ensurePasswordHashes(); err != nil {
		panic(err)
	}

	router.POST("/login", h.login)
	router.POST("/register", h.requestRegistrationCode)
	router.POST("/register/request-code", h.requestRegistrationCode)
	router.POST("/register/verify-code", h.verifyRegistrationCode)
	router.POST("/password-reset/request-code", h.requestPasswordResetCode)
	router.POST("/password-reset/verify-code", h.verifyPasswordResetCode)
	router.POST("/password-reset/confirm", h.completePasswordReset)
	router.POST("/logout", middleware.AuthRequired(db), h.logout)
	router.GET("/me", middleware.AuthRequired(db), h.me)
	router.POST("/change-password", middleware.AuthRequired(db), h.changePassword)
}

func (h *Handler) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)

	var user loginUser
	var passwordHash string
	err := h.db.QueryRow(`
		SELECT id,
		       first_name,
		       last_name,
		       middle_name,
		       full_name,
		       username,
		       COALESCE(email, ''),
		       COALESCE(position, ''),
		       role,
		       COALESCE(password_hash, '')
		FROM users
		WHERE LOWER(username) = LOWER($1)
		LIMIT 1
	`, username).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.MiddleName,
		&user.FullName,
		&user.Username,
		&user.Email,
		&user.Position,
		&user.Role,
		&passwordHash,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(401, errorResponse{Error: "invalid username or password"})
			return
		}
		c.JSON(500, errorResponse{Error: "failed to load user"})
		return
	}

	if !verifyPasswordHash(passwordHash, password) {
		c.JSON(401, errorResponse{Error: "invalid username or password"})
		return
	}

	token, err := generateToken()
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to generate token"})
		return
	}

	if err := h.cleanupExpiredSessions(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to cleanup expired sessions"})
		return
	}

	if _, err := h.db.Exec(`DELETE FROM user_sessions WHERE user_id = $1`, user.ID); err != nil {
		c.JSON(500, errorResponse{Error: "failed to reset old sessions"})
		return
	}

	if _, err := h.db.Exec(`
		INSERT INTO user_sessions (user_id, token, created_at, expires_at)
		VALUES ($1, $2, NOW(), NOW() + ($3 * INTERVAL '1 second'))
	`, user.ID, token, h.sessionTTLSeconds); err != nil {
		c.JSON(500, errorResponse{Error: "failed to create session"})
		return
	}

	c.JSON(200, loginResponse{
		Token: token,
		User:  user,
	})
}

func (h *Handler) requestRegistrationCode(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	firstName := strings.TrimSpace(req.FirstName)
	lastName := strings.TrimSpace(req.LastName)
	middleName := strings.TrimSpace(req.MiddleName)
	position := strings.TrimSpace(req.Position)
	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)
	confirmPassword := strings.TrimSpace(req.ConfirmPassword)

	email, err := validateEmailAddress(req.Email)
	if err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	if firstName == "" || lastName == "" || position == "" || username == "" {
		c.JSON(400, errorResponse{Error: "first_name, last_name, position, email and username are required"})
		return
	}

	if password != confirmPassword {
		c.JSON(400, errorResponse{Error: "password and confirm_password do not match"})
		return
	}

	if err := validatePassword(password); err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	if err := h.ensureEmailDeliveryConfigured(); err != nil {
		c.JSON(500, errorResponse{Error: err.Error()})
		return
	}

	if err := h.assertUserIdentityAvailable(username, email); err != nil {
		switch {
		case errors.Is(err, errUserExists):
			c.JSON(409, errorResponse{Error: "username already exists"})
		case errors.Is(err, errEmailExists):
			c.JSON(409, errorResponse{Error: "email already exists"})
		default:
			c.JSON(500, errorResponse{Error: "failed to validate account identity"})
		}
		return
	}

	pendingStatus, err := h.lookupRegistrationRequestStatus(username, email)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to validate registration request"})
		return
	}
	if pendingStatus == "pending" {
		c.JSON(409, errorResponse{Error: errPendingRequestExists.Error()})
		return
	}
	if pendingStatus == "approved" {
		c.JSON(409, errorResponse{Error: "registration request already approved"})
		return
	}

	passwordHash, err := hashPassword(password)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to secure password"})
		return
	}

	payload, err := json.Marshal(registrationCodePayload{
		FirstName:    firstName,
		LastName:     lastName,
		MiddleName:   middleName,
		Position:     position,
		Email:        email,
		Username:     username,
		PasswordHash: passwordHash,
		Role:         "prorector",
	})
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to prepare verification payload"})
		return
	}

	code, err := h.createVerificationCode("register", email, payload)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to create verification code"})
		return
	}

	if err := h.sendVerificationCodeEmail(email, code, false); err != nil {
		c.JSON(500, errorResponse{Error: err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"message": "verification code sent to email",
		"email":   email,
	})
}

func (h *Handler) verifyRegistrationCode(c *gin.Context) {
	var req verifyRegisterCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	email, err := validateEmailAddress(req.Email)
	if err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	payloadRaw, err := h.consumeVerificationCode("register", email, strings.TrimSpace(req.Code))
	if err != nil {
		if errors.Is(err, errInvalidVerification) {
			c.JSON(400, errorResponse{Error: err.Error()})
			return
		}
		c.JSON(500, errorResponse{Error: "failed to verify code"})
		return
	}

	var payload registrationCodePayload
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		c.JSON(500, errorResponse{Error: "failed to parse verification payload"})
		return
	}

	if err := h.assertUserIdentityAvailable(payload.Username, payload.Email); err != nil {
		switch {
		case errors.Is(err, errUserExists):
			c.JSON(409, errorResponse{Error: "username already exists"})
		case errors.Is(err, errEmailExists):
			c.JSON(409, errorResponse{Error: "email already exists"})
		default:
			c.JSON(500, errorResponse{Error: "failed to validate account identity"})
		}
		return
	}

	if err := h.upsertRegistrationRequest(payload); err != nil {
		c.JSON(500, errorResponse{Error: "failed to create registration request"})
		return
	}

	c.JSON(200, gin.H{
		"message": "registration request sent, wait for admin approval",
	})
}

func (h *Handler) requestPasswordResetCode(c *gin.Context) {
	var req requestPasswordResetCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	email, err := validateEmailAddress(req.Email)
	if err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	if err := h.ensureEmailDeliveryConfigured(); err != nil {
		c.JSON(500, errorResponse{Error: err.Error()})
		return
	}

	var userID int64
	err = h.db.QueryRow(`
		SELECT id
		FROM users
		WHERE LOWER(email) = LOWER($1)
		LIMIT 1
	`, email).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(404, errorResponse{Error: errPasswordResetNotFound.Error()})
			return
		}
		c.JSON(500, errorResponse{Error: "failed to resolve account"})
		return
	}

	payload, err := json.Marshal(passwordResetCodePayload{UserID: userID})
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to prepare reset payload"})
		return
	}

	code, err := h.createVerificationCode("password_reset", email, payload)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to create verification code"})
		return
	}

	if err := h.sendVerificationCodeEmail(email, code, true); err != nil {
		c.JSON(500, errorResponse{Error: err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"message": "verification code sent to email",
		"email":   email,
	})
}

func (h *Handler) verifyPasswordResetCode(c *gin.Context) {
	var req verifyPasswordResetCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	email, err := validateEmailAddress(req.Email)
	if err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	payloadRaw, err := h.consumeVerificationCode("password_reset", email, strings.TrimSpace(req.Code))
	if err != nil {
		if errors.Is(err, errInvalidVerification) {
			c.JSON(400, errorResponse{Error: err.Error()})
			return
		}
		c.JSON(500, errorResponse{Error: "failed to verify code"})
		return
	}

	var payload passwordResetCodePayload
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		c.JSON(500, errorResponse{Error: "failed to parse verification payload"})
		return
	}

	resetToken, err := generateToken()
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to create reset token"})
		return
	}

	if _, err := h.db.Exec(`
		DELETE FROM password_reset_sessions
		WHERE user_id = $1
		  AND used_at IS NULL
	`, payload.UserID); err != nil {
		c.JSON(500, errorResponse{Error: "failed to reset previous password-reset sessions"})
		return
	}

	if _, err := h.db.Exec(`
		INSERT INTO password_reset_sessions (user_id, token, expires_at, created_at, updated_at)
		VALUES ($1, $2, NOW() + ($3 * INTERVAL '1 minute'), NOW(), NOW())
	`, payload.UserID, resetToken, h.resetSessionTTLMinutes); err != nil {
		c.JSON(500, errorResponse{Error: "failed to create password-reset session"})
		return
	}

	c.JSON(200, gin.H{
		"message":     "verification code accepted",
		"reset_token": resetToken,
	})
}

func (h *Handler) completePasswordReset(c *gin.Context) {
	var req completePasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	resetToken := strings.TrimSpace(req.ResetToken)
	newPassword := strings.TrimSpace(req.NewPassword)
	confirmPassword := strings.TrimSpace(req.ConfirmPassword)

	if newPassword != confirmPassword {
		c.JSON(400, errorResponse{Error: "new password and confirmation do not match"})
		return
	}

	if err := validatePassword(newPassword); err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	var sessionID int64
	var userID int64
	err := h.db.QueryRow(`
		SELECT id, user_id
		FROM password_reset_sessions
		WHERE token = $1
		  AND used_at IS NULL
		  AND expires_at > NOW()
		LIMIT 1
	`, resetToken).Scan(&sessionID, &userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(400, errorResponse{Error: "reset session is invalid or expired"})
			return
		}
		c.JSON(500, errorResponse{Error: "failed to load password-reset session"})
		return
	}

	passwordHash, err := hashPassword(newPassword)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to secure password"})
		return
	}

	if _, err := h.db.Exec(`
		UPDATE users
		SET password_hash = $1,
		    updated_at = NOW()
		WHERE id = $2
	`, passwordHash, userID); err != nil {
		c.JSON(500, errorResponse{Error: "failed to update password"})
		return
	}

	if _, err := h.db.Exec(`
		UPDATE password_reset_sessions
		SET used_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1
	`, sessionID); err != nil {
		c.JSON(500, errorResponse{Error: "failed to close password-reset session"})
		return
	}

	if _, err := h.db.Exec(`
		DELETE FROM user_sessions
		WHERE user_id = $1
	`, userID); err != nil {
		c.JSON(500, errorResponse{Error: "failed to revoke active sessions"})
		return
	}

	c.JSON(200, gin.H{"message": "password reset completed successfully"})
}

func (h *Handler) logout(c *gin.Context) {
	token, err := middleware.ExtractBearerToken(c.GetHeader("Authorization"))
	if err != nil {
		c.JSON(401, errorResponse{Error: err.Error()})
		return
	}

	if _, err := h.db.Exec(`
		DELETE FROM user_sessions
		WHERE token = $1
	`, token); err != nil {
		c.JSON(500, errorResponse{Error: "failed to revoke session"})
		return
	}

	c.JSON(200, gin.H{"message": "logged out successfully"})
}

func (h *Handler) me(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(401, errorResponse{Error: "unauthorized"})
		return
	}

	c.JSON(200, meResponse{
		User: meUser{
			ID:         user.ID,
			FirstName:  user.FirstName,
			LastName:   user.LastName,
			MiddleName: user.MiddleName,
			FullName:   user.FullName,
			Username:   user.Username,
			Email:      user.Email,
			Position:   user.Position,
			Role:       user.Role,
		},
	})
}

func (h *Handler) changePassword(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(401, errorResponse{Error: "unauthorized"})
		return
	}

	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	oldPassword := strings.TrimSpace(req.OldPassword)
	newPassword := strings.TrimSpace(req.NewPassword)
	confirmPassword := strings.TrimSpace(req.ConfirmPassword)

	var passwordHash string
	err := h.db.QueryRow(`
		SELECT COALESCE(password_hash, '')
		FROM users
		WHERE id = $1
	`, user.ID).Scan(&passwordHash)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to load current password"})
		return
	}

	if !verifyPasswordHash(passwordHash, oldPassword) {
		c.JSON(400, errorResponse{Error: "old password does not match"})
		return
	}

	if newPassword != confirmPassword {
		c.JSON(400, errorResponse{Error: "new password and confirmation do not match"})
		return
	}

	if err := validatePassword(newPassword); err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	newHash, err := hashPassword(newPassword)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to secure password"})
		return
	}

	if _, err := h.db.Exec(`
		UPDATE users
		SET password_hash = $1,
		    updated_at = NOW()
		WHERE id = $2
	`, newHash, user.ID); err != nil {
		c.JSON(500, errorResponse{Error: "failed to update password"})
		return
	}

	if _, err := h.db.Exec(`
		DELETE FROM user_sessions
		WHERE user_id = $1
	`, user.ID); err != nil {
		c.JSON(500, errorResponse{Error: "failed to revoke active sessions"})
		return
	}

	c.JSON(200, gin.H{"message": "password changed successfully, please log in again"})
}

func normalizeRole(role string) string {
	switch strings.TrimSpace(strings.ToLower(role)) {
	case "admin":
		return "admin"
	case "prorector":
		return "prorector"
	case "viewer":
		return "viewer"
	default:
		return ""
	}
}

func composeFullName(lastName, firstName, middleName string) string {
	parts := []string{
		strings.TrimSpace(lastName),
		strings.TrimSpace(firstName),
		strings.TrimSpace(middleName),
	}
	joined := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			joined = append(joined, part)
		}
	}
	return strings.Join(joined, " ")
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	if !uppercaseRegex.MatchString(password) {
		return fmt.Errorf("password must contain at least one uppercase latin letter")
	}
	if !lowercaseRegex.MatchString(password) {
		return fmt.Errorf("password must contain at least one lowercase latin letter")
	}
	if !digitRegex.MatchString(password) {
		return fmt.Errorf("password must contain at least one digit")
	}
	return nil
}

func hashPassword(password string) (string, error) {
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashBytes), nil
}

func verifyPasswordHash(passwordHash, candidate string) bool {
	trimmedHash := strings.TrimSpace(passwordHash)
	if trimmedHash == "" || strings.TrimSpace(candidate) == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(trimmedHash), []byte(candidate)) == nil
}

func validateEmailAddress(email string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" {
		return "", fmt.Errorf("email is required")
	}

	parsed, err := mail.ParseAddress(normalized)
	if err != nil || strings.ToLower(parsed.Address) != normalized {
		return "", fmt.Errorf("email format is invalid")
	}

	return normalized, nil
}

func generateToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

func generateVerificationCode() (string, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", value.Int64()), nil
}

func (h *Handler) createVerificationCode(purpose, email string, payload []byte) (string, error) {
	if _, err := h.db.Exec(`
		UPDATE email_verification_codes
		SET consumed_at = NOW(),
		    updated_at = NOW()
		WHERE LOWER(email) = LOWER($1)
		  AND purpose = $2
		  AND consumed_at IS NULL
	`, email, purpose); err != nil {
		return "", err
	}

	code, err := generateVerificationCode()
	if err != nil {
		return "", err
	}

	codeHash, err := hashPassword(code)
	if err != nil {
		return "", err
	}

	if _, err := h.db.Exec(`
		INSERT INTO email_verification_codes (purpose, email, code_hash, payload, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4::jsonb, NOW() + ($5 * INTERVAL '1 minute'), NOW(), NOW())
	`, purpose, email, codeHash, string(payload), h.otpTTLMinutes); err != nil {
		return "", err
	}

	return code, nil
}

func (h *Handler) consumeVerificationCode(purpose, email, code string) ([]byte, error) {
	var id int64
	var codeHash string
	var payload []byte
	err := h.db.QueryRow(`
		SELECT id, code_hash, payload
		FROM email_verification_codes
		WHERE LOWER(email) = LOWER($1)
		  AND purpose = $2
		  AND consumed_at IS NULL
		  AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`, email, purpose).Scan(&id, &codeHash, &payload)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errInvalidVerification
		}
		return nil, err
	}

	if !verifyPasswordHash(codeHash, strings.TrimSpace(code)) {
		return nil, errInvalidVerification
	}

	if _, err := h.db.Exec(`
		UPDATE email_verification_codes
		SET consumed_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1
	`, id); err != nil {
		return nil, err
	}

	return payload, nil
}

func (h *Handler) upsertRegistrationRequest(payload registrationCodePayload) error {
	var existingID int64
	var existingStatus string
	err := h.db.QueryRow(`
		SELECT id, status
		FROM registration_requests
		WHERE LOWER(email) = LOWER($1)
		   OR LOWER(username) = LOWER($2)
		ORDER BY id DESC
		LIMIT 1
	`, payload.Email, payload.Username).Scan(&existingID, &existingStatus)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	fullName := composeFullName(payload.LastName, payload.FirstName, payload.MiddleName)

	if errors.Is(err, sql.ErrNoRows) {
		_, err = h.db.Exec(`
			INSERT INTO registration_requests (
				first_name,
				last_name,
				middle_name,
				full_name,
				username,
				email,
				position,
				password_hash,
				role,
				status,
				rejection_reason,
				email_verified_at,
				created_at,
				updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'pending', '', NOW(), NOW(), NOW())
		`, payload.FirstName, payload.LastName, payload.MiddleName, fullName, payload.Username, payload.Email, payload.Position, payload.PasswordHash, payload.Role)
		return err
	}

	if existingStatus == "approved" {
		return fmt.Errorf("registration request already approved")
	}

	_, err = h.db.Exec(`
		UPDATE registration_requests
		SET first_name = $1,
		    last_name = $2,
		    middle_name = $3,
		    full_name = $4,
		    username = $5,
		    email = $6,
		    position = $7,
		    password_hash = $8,
		    role = $9,
		    status = 'pending',
		    rejection_reason = '',
		    reviewed_by = NULL,
		    reviewed_at = NULL,
		    approved_user_id = NULL,
		    email_verified_at = NOW(),
		    updated_at = NOW()
		WHERE id = $10
	`, payload.FirstName, payload.LastName, payload.MiddleName, fullName, payload.Username, payload.Email, payload.Position, payload.PasswordHash, payload.Role, existingID)
	return err
}

func (h *Handler) lookupRegistrationRequestStatus(username, email string) (string, error) {
	var status string
	err := h.db.QueryRow(`
		SELECT status
		FROM registration_requests
		WHERE LOWER(username) = LOWER($1)
		   OR LOWER(email) = LOWER($2)
		ORDER BY id DESC
		LIMIT 1
	`, username, email).Scan(&status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(strings.ToLower(status)), nil
}

func (h *Handler) assertUserIdentityAvailable(username, email string) error {
	var existingID int64
	args := []any{username}
	query := `
		SELECT id
		FROM users
		WHERE LOWER(username) = LOWER($1)
	`
	if strings.TrimSpace(email) != "" {
		args = append(args, email)
		query += ` OR LOWER(email) = LOWER($2)`
	}
	query += ` LIMIT 1`

	err := h.db.QueryRow(query, args...).Scan(&existingID)
	if err == nil {
		if strings.TrimSpace(email) != "" {
			var byUsername bool
			_ = h.db.QueryRow(`
				SELECT EXISTS (
					SELECT 1
					FROM users
					WHERE LOWER(username) = LOWER($1)
				)
			`, username).Scan(&byUsername)
			if byUsername {
				return errUserExists
			}
			return errEmailExists
		}
		return errUserExists
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	return nil
}

func (h *Handler) ensureEmailDeliveryConfigured() error {
	fromEmail := strings.TrimSpace(h.smtpFromEmail)
	if fromEmail == "" {
		fromEmail = strings.TrimSpace(h.smtpUsername)
	}

	if strings.TrimSpace(h.smtpHost) == "" ||
		strings.TrimSpace(h.smtpPort) == "" ||
		strings.TrimSpace(h.smtpUsername) == "" ||
		strings.TrimSpace(h.smtpPassword) == "" ||
		fromEmail == "" {
		return fmt.Errorf("%w: set SMTP_HOST, SMTP_PORT, SMTP_USERNAME, SMTP_PASSWORD and SMTP_FROM_EMAIL in .env", errEmailNotConfigured)
	}

	return nil
}

func (h *Handler) sendVerificationCodeEmail(toEmail, code string, isPasswordReset bool) error {
	if err := h.ensureEmailDeliveryConfigured(); err != nil {
		return err
	}

	subject := "Oper Plan verification code"
	intro := "Use this code to continue registration."
	if isPasswordReset {
		subject = "Oper Plan password reset code"
		intro = "Use this code to reset your password."
	}

	fromEmail := strings.TrimSpace(h.smtpFromEmail)
	if fromEmail == "" {
		fromEmail = strings.TrimSpace(h.smtpUsername)
	}
	fromName := strings.TrimSpace(h.smtpFromName)
	if fromName == "" {
		fromName = "Oper Plan"
	}

	body := strings.Join([]string{
		intro,
		"",
		fmt.Sprintf("Verification code: %s", code),
		fmt.Sprintf("This code expires in %d minutes.", h.otpTTLMinutes),
		"",
		"If you did not request this action, ignore this email.",
	}, "\r\n")

	message := strings.Join([]string{
		fmt.Sprintf("From: %s <%s>", fromName, fromEmail),
		fmt.Sprintf("To: %s", toEmail),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		`Content-Type: text/plain; charset="UTF-8"`,
		"",
		body,
	}, "\r\n")

	auth := smtp.PlainAuth("", h.smtpUsername, h.smtpPassword, h.smtpHost)
	if err := smtp.SendMail(h.smtpHost+":"+h.smtpPort, auth, fromEmail, []string{toEmail}, []byte(message)); err != nil {
		return fmt.Errorf("failed to send verification code email: %w", err)
	}

	return nil
}

func normalizeBootstrapAdminUsername(username string) string {
	trimmed := strings.TrimSpace(username)
	if trimmed == "" {
		return "admin"
	}
	return trimmed
}

func (h *Handler) cleanupExpiredSessions() error {
	_, err := h.db.Exec(`
		DELETE FROM user_sessions
		WHERE expires_at <= NOW()
	`)
	return err
}

func (h *Handler) ensureUsersTable() error {
	baseStatements := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id BIGSERIAL PRIMARY KEY,
			first_name VARCHAR(128) NOT NULL DEFAULT '',
			last_name VARCHAR(128) NOT NULL DEFAULT '',
			middle_name VARCHAR(128) NOT NULL DEFAULT '',
			full_name VARCHAR(255) NOT NULL DEFAULT '',
			username VARCHAR(64) UNIQUE,
			email VARCHAR(255) NOT NULL DEFAULT '',
			position VARCHAR(255) NOT NULL DEFAULT '',
			password_hash VARCHAR(255) NOT NULL DEFAULT '',
			role VARCHAR(32) NOT NULL DEFAULT 'viewer',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS username VARCHAR(64);`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS email VARCHAR(255) NOT NULL DEFAULT '';`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS position VARCHAR(255) NOT NULL DEFAULT '';`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255) NOT NULL DEFAULT '';`,
		`DROP INDEX IF EXISTS idx_users_email;`,
		`UPDATE users SET email = '' WHERE email IS NULL;`,
		`UPDATE users SET position = '' WHERE position IS NULL;`,
		`UPDATE users SET password_hash = '' WHERE password_hash IS NULL;`,
		`CREATE UNIQUE INDEX IF NOT EXISTS users_username_uindex ON users (username);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS users_email_uindex ON users ((LOWER(email))) WHERE COALESCE(TRIM(email), '') <> '';`,
		`CREATE TABLE IF NOT EXISTS user_sessions (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token VARCHAR(255) NOT NULL UNIQUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			expires_at TIMESTAMPTZ NOT NULL
		);`,
		`ALTER TABLE user_sessions ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;`,
	}

	for _, statement := range baseStatements {
		if _, err := h.db.Exec(statement); err != nil {
			return err
		}
	}

	if _, err := h.db.Exec(`
		UPDATE user_sessions
		SET expires_at = created_at + ($1 * INTERVAL '1 second')
		WHERE expires_at IS NULL
	`, h.sessionTTLSeconds); err != nil {
		return err
	}

	authTablesStatements := []string{
		`ALTER TABLE user_sessions ALTER COLUMN expires_at SET NOT NULL;`,
		`CREATE TABLE IF NOT EXISTS registration_requests (
			id BIGSERIAL PRIMARY KEY,
			first_name VARCHAR(128) NOT NULL,
			last_name VARCHAR(128) NOT NULL,
			middle_name VARCHAR(128) NOT NULL DEFAULT '',
			full_name VARCHAR(255) NOT NULL,
			username VARCHAR(64) NOT NULL,
			email VARCHAR(255) NOT NULL,
			position VARCHAR(255) NOT NULL DEFAULT '',
			password_hash VARCHAR(255) NOT NULL,
			role VARCHAR(32) NOT NULL DEFAULT 'prorector',
			status VARCHAR(32) NOT NULL DEFAULT 'pending',
			rejection_reason TEXT NOT NULL DEFAULT '',
			reviewed_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
			reviewed_at TIMESTAMPTZ NULL,
			approved_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
			email_verified_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS registration_requests_status_idx ON registration_requests (status);`,
		`CREATE INDEX IF NOT EXISTS registration_requests_email_idx ON registration_requests ((LOWER(email)));`,
		`CREATE INDEX IF NOT EXISTS registration_requests_username_idx ON registration_requests ((LOWER(username)));`,
		`CREATE TABLE IF NOT EXISTS email_verification_codes (
			id BIGSERIAL PRIMARY KEY,
			purpose VARCHAR(32) NOT NULL,
			email VARCHAR(255) NOT NULL,
			code_hash VARCHAR(255) NOT NULL,
			payload JSONB NOT NULL DEFAULT '{}'::jsonb,
			expires_at TIMESTAMPTZ NOT NULL,
			consumed_at TIMESTAMPTZ NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS email_verification_codes_lookup_idx ON email_verification_codes ((LOWER(email)), purpose, created_at DESC);`,
		`CREATE TABLE IF NOT EXISTS password_reset_sessions (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token VARCHAR(128) NOT NULL UNIQUE,
			expires_at TIMESTAMPTZ NOT NULL,
			used_at TIMESTAMPTZ NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS password_reset_sessions_user_idx ON password_reset_sessions (user_id);`,
	}

	for _, statement := range authTablesStatements {
		if _, err := h.db.Exec(statement); err != nil {
			return err
		}
	}

	return nil
}

func (h *Handler) ensureBootstrapAdmin() error {
	if h.bootstrapAdminPassword == "" {
		slog.Info("bootstrap admin skipped: BOOTSTRAP_ADMIN_PASSWORD is empty")
		return nil
	}

	if err := validatePassword(h.bootstrapAdminPassword); err != nil {
		return fmt.Errorf("invalid bootstrap admin password: %w", err)
	}

	var existingID int64
	err := h.db.QueryRow(`
		SELECT id
		FROM users
		WHERE LOWER(username) = LOWER($1)
		LIMIT 1
	`, h.bootstrapAdminUsername).Scan(&existingID)
	if err == nil {
		slog.Info("bootstrap admin skipped: user already exists", "username", h.bootstrapAdminUsername)
		return nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	_, err = h.createUser(
		"System",
		"Admin",
		"",
		"Administrator",
		"",
		h.bootstrapAdminUsername,
		h.bootstrapAdminPassword,
		h.bootstrapAdminPassword,
		"admin",
	)
	if err != nil {
		return err
	}

	slog.Warn("bootstrap admin created; rotate password after first login", "username", h.bootstrapAdminUsername)
	return nil
}

func (h *Handler) ensurePasswordHashes() error {
	hasLegacyColumn, err := h.columnExists("users", "password_plain")
	if err != nil {
		return err
	}
	if !hasLegacyColumn {
		return nil
	}

	rows, err := h.db.Query(`
		SELECT id, COALESCE(password_plain, '')
		FROM users
		WHERE COALESCE(TRIM(password_hash), '') = ''
		  AND COALESCE(TRIM(password_plain), '') <> ''
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type candidate struct {
		id       int64
		password string
	}
	toUpdate := make([]candidate, 0)
	for rows.Next() {
		var item candidate
		if err := rows.Scan(&item.id, &item.password); err != nil {
			return err
		}
		item.password = strings.TrimSpace(item.password)
		if item.password != "" {
			toUpdate = append(toUpdate, item)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, item := range toUpdate {
		passwordHash, err := hashPassword(item.password)
		if err != nil {
			return err
		}
		if _, err := h.db.Exec(`
			UPDATE users
			SET password_hash = $1,
			    updated_at = NOW()
			WHERE id = $2
		`, passwordHash, item.id); err != nil {
			return err
		}
	}

	if _, err := h.db.Exec(`ALTER TABLE users DROP COLUMN IF EXISTS password_plain`); err != nil {
		return err
	}

	return nil
}

func (h *Handler) columnExists(tableName, columnName string) (bool, error) {
	var exists bool
	err := h.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = 'public'
			  AND table_name = $1
			  AND column_name = $2
		)
	`, tableName, columnName).Scan(&exists)
	return exists, err
}

func (h *Handler) createUser(firstName, lastName, middleName, position, email, username, password, confirmPassword, role string) (loginUser, error) {
	if firstName == "" || lastName == "" || position == "" || username == "" {
		return loginUser{}, fmt.Errorf("first_name, last_name, position and username are required")
	}
	if password != confirmPassword {
		return loginUser{}, fmt.Errorf("password and confirm_password do not match")
	}
	if err := validatePassword(password); err != nil {
		return loginUser{}, fmt.Errorf("%w: %s", errPasswordValidation, err.Error())
	}

	role = normalizeRole(role)
	if role == "" {
		return loginUser{}, errInvalidRole
	}

	normalizedEmail := ""
	if strings.TrimSpace(email) != "" {
		var err error
		normalizedEmail, err = validateEmailAddress(email)
		if err != nil {
			return loginUser{}, err
		}
	}

	if err := h.assertUserIdentityAvailable(username, normalizedEmail); err != nil {
		return loginUser{}, err
	}

	passwordHash, err := hashPassword(password)
	if err != nil {
		return loginUser{}, fmt.Errorf("failed to secure password")
	}

	fullName := composeFullName(lastName, firstName, middleName)

	var user loginUser
	err = h.db.QueryRow(`
		INSERT INTO users (
			first_name,
			last_name,
			middle_name,
			full_name,
			username,
			email,
			position,
			password_hash,
			role,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		RETURNING id, first_name, last_name, middle_name, full_name, username, COALESCE(email, ''), COALESCE(position, ''), role
	`, firstName, lastName, middleName, fullName, username, normalizedEmail, position, passwordHash, role).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.MiddleName,
		&user.FullName,
		&user.Username,
		&user.Email,
		&user.Position,
		&user.Role,
	)
	if err != nil {
		return loginUser{}, err
	}

	return user, nil
}

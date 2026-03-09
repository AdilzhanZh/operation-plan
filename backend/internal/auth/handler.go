package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"OperationPlan/internal/middleware"

	"github.com/gin-gonic/gin"
)

var (
	uppercaseRegex = regexp.MustCompile(`[A-Z]`)
	lowercaseRegex = regexp.MustCompile(`[a-z]`)
	digitRegex     = regexp.MustCompile(`[0-9]`)
)

type Handler struct {
	db *sql.DB
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type registerRequest struct {
	FirstName       string `json:"first_name" binding:"required"`
	LastName        string `json:"last_name" binding:"required"`
	MiddleName      string `json:"middle_name"`
	Username        string `json:"username" binding:"required"`
	Password        string `json:"password" binding:"required"`
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
	Role       string `json:"role"`
}

func RegisterRoutes(router gin.IRoutes, db *sql.DB) {
	h := &Handler{db: db}

	if err := h.ensureUsersTable(); err != nil {
		panic(err)
	}
	if err := h.ensureDefaultAdmin(); err != nil {
		panic(err)
	}

	router.POST("/login", h.login)
	router.POST("/register", h.register)
	router.GET("/me", middleware.AuthRequired(db), h.me)
	router.POST("/change-password", middleware.AuthRequired(db), h.changePassword)
}

// login godoc
// @Summary Login
// @Description Authenticates user by username and password
// @Tags auth
// @Accept json
// @Produce json
// @Param payload body loginRequest true "Login payload"
// @Success 200 {object} loginResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /login [post]
func (h *Handler) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)

	var user loginUser
	var storedPassword string
	err := h.db.QueryRow(`
		SELECT id, first_name, last_name, middle_name, full_name, username, role, password_plain
		FROM users
		WHERE username = $1
	`, username).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.MiddleName,
		&user.FullName,
		&user.Username,
		&user.Role,
		&storedPassword,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(401, errorResponse{Error: "invalid username or password"})
			return
		}
		c.JSON(500, errorResponse{Error: "failed to load user"})
		return
	}

	if storedPassword != password {
		c.JSON(401, errorResponse{Error: "invalid username or password"})
		return
	}

	token, err := generateToken()
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to generate token"})
		return
	}

	if _, err := h.db.Exec(`DELETE FROM user_sessions WHERE user_id = $1`, user.ID); err != nil {
		c.JSON(500, errorResponse{Error: "failed to reset old sessions"})
		return
	}

	if _, err := h.db.Exec(`
		INSERT INTO user_sessions (user_id, token, created_at)
		VALUES ($1, $2, NOW())
	`, user.ID, token); err != nil {
		c.JSON(500, errorResponse{Error: "failed to create session"})
		return
	}

	c.JSON(200, loginResponse{
		Token: token,
		User:  user,
	})
}

// register godoc
// @Summary Register
// @Description Registers new user with default role "viewer"
// @Tags auth
// @Accept json
// @Produce json
// @Param payload body registerRequest true "Register payload"
// @Success 201 {object} loginUser
// @Failure 400 {object} errorResponse
// @Failure 409 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /register [post]
func (h *Handler) register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	user, err := h.createUser(
		strings.TrimSpace(req.FirstName),
		strings.TrimSpace(req.LastName),
		strings.TrimSpace(req.MiddleName),
		strings.TrimSpace(req.Username),
		strings.TrimSpace(req.Password),
		strings.TrimSpace(req.ConfirmPassword),
		"viewer",
	)
	if err != nil {
		if errors.Is(err, errUserExists) {
			c.JSON(409, errorResponse{Error: "username already exists"})
			return
		}
		if errors.Is(err, errInvalidRole) || errors.Is(err, errPasswordValidation) {
			c.JSON(400, errorResponse{Error: err.Error()})
			return
		}
		c.JSON(500, errorResponse{Error: "failed to register user"})
		return
	}

	c.JSON(201, user)
}

// me godoc
// @Summary Current user profile
// @Description Returns authenticated user profile
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} meResponse
// @Failure 401 {object} errorResponse
// @Router /me [get]
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
			Role:       user.Role,
		},
	})
}

// changePassword godoc
// @Summary Change password
// @Description Changes password for current user
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param payload body changePasswordRequest true "Change password payload"
// @Success 200 {object} gin.H
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /change-password [post]
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

	var storedPassword string
	err := h.db.QueryRow(`SELECT password_plain FROM users WHERE id = $1`, user.ID).Scan(&storedPassword)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to load current password"})
		return
	}

	if storedPassword != oldPassword {
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

	if _, err := h.db.Exec(`
		UPDATE users
		SET password_plain = $1,
		    updated_at = NOW()
		WHERE id = $2
	`, newPassword, user.ID); err != nil {
		c.JSON(500, errorResponse{Error: "failed to update password"})
		return
	}

	c.JSON(200, gin.H{"message": "password changed successfully"})
}

var (
	errUserExists         = errors.New("username already exists")
	errInvalidRole        = errors.New("invalid role")
	errPasswordValidation = errors.New("password validation error")
)

func (h *Handler) createUser(firstName, lastName, middleName, username, password, confirmPassword, role string) (loginUser, error) {
	if firstName == "" || lastName == "" || username == "" {
		return loginUser{}, fmt.Errorf("first_name, last_name and username are required")
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

	var existingID int64
	err := h.db.QueryRow(`SELECT id FROM users WHERE username = $1`, username).Scan(&existingID)
	if err == nil {
		return loginUser{}, errUserExists
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return loginUser{}, err
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
			password_plain,
			role,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING id, first_name, last_name, middle_name, full_name, username, role
	`, firstName, lastName, middleName, fullName, username, password, role).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.MiddleName,
		&user.FullName,
		&user.Username,
		&user.Role,
	)
	if err != nil {
		return loginUser{}, err
	}

	return user, nil
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

func generateToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

func (h *Handler) ensureUsersTable() error {
	if _, err := h.db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id BIGSERIAL PRIMARY KEY,
			first_name VARCHAR(128) NOT NULL DEFAULT '',
			last_name VARCHAR(128) NOT NULL DEFAULT '',
			middle_name VARCHAR(128) NOT NULL DEFAULT '',
			full_name VARCHAR(255) NOT NULL DEFAULT '',
			username VARCHAR(64) UNIQUE,
			password_plain VARCHAR(255) NOT NULL DEFAULT '',
			role VARCHAR(32) NOT NULL DEFAULT 'viewer',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`); err != nil {
		return err
	}

	compatStatements := []string{
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS username VARCHAR(64);`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS email VARCHAR(255);`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255);`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS password_plain VARCHAR(255) NOT NULL DEFAULT '';`,
		`ALTER TABLE users ALTER COLUMN email DROP NOT NULL;`,
		`ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;`,
	}
	for _, stmt := range compatStatements {
		if _, err := h.db.Exec(stmt); err != nil {
			return err
		}
	}

	if _, err := h.db.Exec(`
		CREATE TABLE IF NOT EXISTS user_sessions (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token VARCHAR(255) NOT NULL UNIQUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`); err != nil {
		return err
	}

	if _, err := h.db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS users_username_uindex ON users (username)`); err != nil {
		return err
	}

	return nil
}

func (h *Handler) ensureDefaultAdmin() error {
	var adminID int64
	err := h.db.QueryRow(`SELECT id FROM users WHERE username = 'admin' LIMIT 1`).Scan(&adminID)
	if errors.Is(err, sql.ErrNoRows) {
		_, createErr := h.createUser("System", "Admin", "", "admin", "Admin123", "Admin123", "admin")
		return createErr
	}
	if err != nil {
		return err
	}

	_, err = h.db.Exec(`
		UPDATE users
		SET role = 'admin',
		    first_name = CASE WHEN TRIM(first_name) = '' THEN 'System' ELSE first_name END,
		    last_name = CASE WHEN TRIM(last_name) = '' THEN 'Admin' ELSE last_name END,
		    full_name = CASE
		      WHEN TRIM(full_name) = '' THEN CONCAT(
		        CASE WHEN TRIM(last_name) = '' THEN 'Admin' ELSE last_name END,
		        ' ',
		        CASE WHEN TRIM(first_name) = '' THEN 'System' ELSE first_name END
		      )
		      ELSE full_name
		    END,
		    password_plain = 'Admin123',
		    updated_at = NOW()
		WHERE id = $1
	`, adminID)
	return err
}

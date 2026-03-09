package user

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

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

type listUsersResponse struct {
	Items []UserResponse `json:"items"`
}

type UserResponse struct {
	ID            int64     `json:"id"`
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
	MiddleName    string    `json:"middle_name"`
	FullName      string    `json:"full_name"`
	Username      string    `json:"username"`
	PasswordPlain string    `json:"password_plain"`
	Role          string    `json:"role"`
	CreatedAt     time.Time `json:"created_at"`
}

type createUserRequest struct {
	FirstName       string `json:"first_name" binding:"required"`
	LastName        string `json:"last_name" binding:"required"`
	MiddleName      string `json:"middle_name"`
	Username        string `json:"username" binding:"required"`
	Password        string `json:"password" binding:"required"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
	Role            string `json:"role" binding:"required"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func RegisterRoutes(router gin.IRoutes, db *sql.DB) {
	h := &Handler{db: db}

	router.GET("/users", h.listUsers)
	router.POST("/users", h.createUser)
}

// listUsers godoc
// @Summary List users
// @Description Returns all users for admin panel (including current plain passwords)
// @Tags users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} listUsersResponse
// @Failure 500 {object} errorResponse
// @Router /users [get]
func (h *Handler) listUsers(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT id, first_name, last_name, middle_name, full_name, username, password_plain, role, created_at
		FROM users
		ORDER BY id ASC
	`)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to load users"})
		return
	}
	defer rows.Close()

	items := make([]UserResponse, 0)
	for rows.Next() {
		var item UserResponse
		if err := rows.Scan(
			&item.ID,
			&item.FirstName,
			&item.LastName,
			&item.MiddleName,
			&item.FullName,
			&item.Username,
			&item.PasswordPlain,
			&item.Role,
			&item.CreatedAt,
		); err != nil {
			c.JSON(500, errorResponse{Error: "failed to parse users"})
			return
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to iterate users"})
		return
	}

	c.JSON(200, listUsersResponse{Items: items})
}

// createUser godoc
// @Summary Create new user
// @Description Admin creates a new user with specified role
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param payload body createUserRequest true "Create user payload"
// @Success 201 {object} UserResponse
// @Failure 400 {object} errorResponse
// @Failure 409 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /users [post]
func (h *Handler) createUser(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	firstName := strings.TrimSpace(req.FirstName)
	lastName := strings.TrimSpace(req.LastName)
	middleName := strings.TrimSpace(req.MiddleName)
	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)
	confirmPassword := strings.TrimSpace(req.ConfirmPassword)
	role := normalizeRole(req.Role)

	if firstName == "" || lastName == "" || username == "" {
		c.JSON(400, errorResponse{Error: "first_name, last_name and username are required"})
		return
	}

	if role == "" {
		c.JSON(400, errorResponse{Error: "role must be one of: admin, prorector, viewer"})
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

	var existsID int64
	err := h.db.QueryRow(`SELECT id FROM users WHERE username = $1`, username).Scan(&existsID)
	if err == nil {
		c.JSON(409, errorResponse{Error: "username already exists"})
		return
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		c.JSON(500, errorResponse{Error: "failed to validate username"})
		return
	}

	fullName := composeFullName(lastName, firstName, middleName)

	var created UserResponse
	err = h.db.QueryRow(`
		INSERT INTO users (
			first_name, last_name, middle_name, full_name,
			username, password_plain, role, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING id, first_name, last_name, middle_name, full_name, username, password_plain, role, created_at
	`, firstName, lastName, middleName, fullName, username, password, role).Scan(
		&created.ID,
		&created.FirstName,
		&created.LastName,
		&created.MiddleName,
		&created.FullName,
		&created.Username,
		&created.PasswordPlain,
		&created.Role,
		&created.CreatedAt,
	)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to create user"})
		return
	}

	c.JSON(201, created)
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
	parts := []string{lastName, firstName, middleName}
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return strings.Join(values, " ")
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

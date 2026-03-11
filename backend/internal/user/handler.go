package user

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strconv"
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
	db *sql.DB
}

type listUsersResponse struct {
	Items []UserResponse `json:"items"`
	Meta  listMeta       `json:"meta"`
}

type listMeta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type prorectorListResponse struct {
	Items []ProrectorOption `json:"items"`
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

type ProrectorOption struct {
	ID       int64  `json:"id"`
	FullName string `json:"full_name"`
	Username string `json:"username"`
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
	router.GET("/users/prorectors", h.listProrectors)
	router.POST("/users", h.createUser)
}

// listUsers godoc
// @Summary List users
// @Description Returns all users for admin panel
// @Tags users
// @Produce json
// @Security BearerAuth
// @Param role query string false "Filter by role: admin,prorector,viewer"
// @Param q query string false "Search by full name / username"
// @Param page query int false "Page number (default 1)"
// @Param limit query int false "Items per page (default 20, max 100)"
// @Success 200 {object} listUsersResponse
// @Failure 500 {object} errorResponse
// @Router /users [get]
func (h *Handler) listUsers(c *gin.Context) {
	page, limit, err := parsePagination(c.Query("page"), c.Query("limit"))
	if err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	roleFilter := strings.TrimSpace(strings.ToLower(c.Query("role")))
	if roleFilter != "" && normalizeRole(roleFilter) == "" {
		c.JSON(400, errorResponse{Error: "invalid role filter"})
		return
	}

	searchQuery := strings.TrimSpace(c.Query("q"))
	where := []string{"1=1"}
	args := make([]any, 0, 4)

	if roleFilter != "" {
		args = append(args, roleFilter)
		where = append(where, fmt.Sprintf("role = $%d", len(args)))
	}

	if searchQuery != "" {
		args = append(args, "%"+searchQuery+"%")
		placeholder := fmt.Sprintf("$%d", len(args))
		where = append(where, fmt.Sprintf("(full_name ILIKE %s OR username ILIKE %s)", placeholder, placeholder))
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	err = h.db.QueryRow(`
		SELECT COUNT(*)
		FROM users
		WHERE `+whereClause, args...).Scan(&total)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to load users"})
		return
	}

	queryArgs := append([]any{}, args...)
	queryArgs = append(queryArgs, limit, (page-1)*limit)
	rows, err := h.db.Query(`
		SELECT id, first_name, last_name, middle_name, full_name, username, role, created_at
		FROM users
		WHERE `+whereClause+`
		ORDER BY id ASC
		LIMIT $`+strconv.Itoa(len(queryArgs)-1)+`
		OFFSET $`+strconv.Itoa(len(queryArgs)), queryArgs...)
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
			&item.Role,
			&item.CreatedAt,
		); err != nil {
			c.JSON(500, errorResponse{Error: "failed to parse users"})
			return
		}
		item.PasswordPlain = "hidden"
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to iterate users"})
		return
	}

	totalPages := total / limit
	if total%limit != 0 {
		totalPages++
	}

	c.JSON(200, listUsersResponse{
		Items: items,
		Meta: listMeta{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// listProrectors godoc
// @Summary List prorector users
// @Description Returns users with role=prorector for assignment in plans
// @Tags users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} prorectorListResponse
// @Failure 500 {object} errorResponse
// @Router /users/prorectors [get]
func (h *Handler) listProrectors(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT id, full_name, username
		FROM users
		WHERE role = 'prorector'
		ORDER BY full_name ASC, id ASC
	`)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to load prorectors"})
		return
	}
	defer rows.Close()

	items := make([]ProrectorOption, 0)
	for rows.Next() {
		var item ProrectorOption
		if err := rows.Scan(&item.ID, &item.FullName, &item.Username); err != nil {
			c.JSON(500, errorResponse{Error: "failed to parse prorectors"})
			return
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to iterate prorectors"})
		return
	}

	c.JSON(200, prorectorListResponse{Items: items})
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
	passwordHash, err := hashPassword(password)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to secure password"})
		return
	}

	var existsID int64
	err = h.db.QueryRow(`SELECT id FROM users WHERE username = $1`, username).Scan(&existsID)
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
			username, password_hash, password_plain, role, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, '', $7, NOW(), NOW())
		RETURNING id, first_name, last_name, middle_name, full_name, username, role, created_at
	`, firstName, lastName, middleName, fullName, username, passwordHash, role).Scan(
		&created.ID,
		&created.FirstName,
		&created.LastName,
		&created.MiddleName,
		&created.FullName,
		&created.Username,
		&created.Role,
		&created.CreatedAt,
	)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to create user"})
		return
	}
	created.PasswordPlain = "hidden"

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

func hashPassword(password string) (string, error) {
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashBytes), nil
}

func parsePagination(pageRaw, limitRaw string) (int, int, error) {
	page := 1
	limit := 20

	if strings.TrimSpace(pageRaw) != "" {
		parsedPage, err := strconv.Atoi(strings.TrimSpace(pageRaw))
		if err != nil || parsedPage <= 0 {
			return 0, 0, fmt.Errorf("page must be positive integer")
		}
		page = parsedPage
	}

	if strings.TrimSpace(limitRaw) != "" {
		parsedLimit, err := strconv.Atoi(strings.TrimSpace(limitRaw))
		if err != nil || parsedLimit <= 0 {
			return 0, 0, fmt.Errorf("limit must be positive integer")
		}
		limit = parsedLimit
	}

	if limit > 100 {
		limit = 100
	}

	return page, limit, nil
}

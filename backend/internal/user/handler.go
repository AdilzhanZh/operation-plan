package user

import (
	"OperationPlan/internal/middleware"
	"database/sql"
	"errors"
	"fmt"
	"net/mail"
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

type listRegistrationRequestsResponse struct {
	Items []RegistrationRequestResponse `json:"items"`
	Meta  listMeta                      `json:"meta"`
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
	ID         int64     `json:"id"`
	FirstName  string    `json:"first_name"`
	LastName   string    `json:"last_name"`
	MiddleName string    `json:"middle_name"`
	FullName   string    `json:"full_name"`
	Username   string    `json:"username"`
	Email      string    `json:"email"`
	Position   string    `json:"position"`
	Role       string    `json:"role"`
	CreatedAt  time.Time `json:"created_at"`
}

type RegistrationRequestResponse struct {
	ID              int64      `json:"id"`
	FirstName       string     `json:"first_name"`
	LastName        string     `json:"last_name"`
	MiddleName      string     `json:"middle_name"`
	FullName        string     `json:"full_name"`
	Username        string     `json:"username"`
	Email           string     `json:"email"`
	Position        string     `json:"position"`
	Role            string     `json:"role"`
	Status          string     `json:"status"`
	RejectionReason string     `json:"rejection_reason"`
	CreatedAt       time.Time  `json:"created_at"`
	ReviewedAt      *time.Time `json:"reviewed_at"`
	ReviewedByName  string     `json:"reviewed_by_name"`
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
	Position        string `json:"position" binding:"required"`
	Email           string `json:"email" binding:"required"`
	Username        string `json:"username" binding:"required"`
	Password        string `json:"password" binding:"required"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
	Role            string `json:"role" binding:"required"`
}

type rejectRegistrationRequestBody struct {
	Reason string `json:"reason"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func RegisterRoutes(router gin.IRoutes, db *sql.DB) {
	h := &Handler{db: db}

	router.GET("/users", h.listUsers)
	router.GET("/users/prorectors", h.listProrectors)
	router.POST("/users", h.createUser)
	router.DELETE("/users/:id", h.deleteUser)
	router.GET("/registration-requests", h.listRegistrationRequests)
	router.PATCH("/registration-requests/:id/approve", h.approveRegistrationRequest)
	router.PATCH("/registration-requests/:id/reject", h.rejectRegistrationRequest)
}

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
		where = append(where, fmt.Sprintf("(full_name ILIKE %s OR username ILIKE %s OR email ILIKE %s OR position ILIKE %s)", placeholder, placeholder, placeholder, placeholder))
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
		SELECT id, first_name, last_name, middle_name, full_name, username, COALESCE(email, ''), COALESCE(position, ''), role, created_at
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
			&item.Email,
			&item.Position,
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

func (h *Handler) listRegistrationRequests(c *gin.Context) {
	page, limit, err := parsePagination(c.Query("page"), c.Query("limit"))
	if err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	statusFilter := strings.TrimSpace(strings.ToLower(c.Query("status")))
	where := []string{"1=1"}
	args := make([]any, 0, 4)

	if statusFilter != "" {
		args = append(args, statusFilter)
		where = append(where, fmt.Sprintf("rr.status = $%d", len(args)))
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	err = h.db.QueryRow(`
		SELECT COUNT(*)
		FROM registration_requests rr
		WHERE `+whereClause, args...).Scan(&total)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to load registration requests"})
		return
	}

	queryArgs := append([]any{}, args...)
	queryArgs = append(queryArgs, limit, (page-1)*limit)
	rows, err := h.db.Query(`
		SELECT rr.id,
		       rr.first_name,
		       rr.last_name,
		       rr.middle_name,
		       rr.full_name,
		       rr.username,
		       rr.email,
		       COALESCE(rr.position, ''),
		       rr.role,
		       rr.status,
		       COALESCE(rr.rejection_reason, ''),
		       rr.created_at,
		       rr.reviewed_at,
		       COALESCE(reviewer.full_name, '')
		FROM registration_requests rr
		LEFT JOIN users reviewer ON reviewer.id = rr.reviewed_by
		WHERE `+whereClause+`
		ORDER BY CASE WHEN rr.status = 'pending' THEN 0 ELSE 1 END, rr.created_at DESC, rr.id DESC
		LIMIT $`+strconv.Itoa(len(queryArgs)-1)+`
		OFFSET $`+strconv.Itoa(len(queryArgs)), queryArgs...)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to load registration requests"})
		return
	}
	defer rows.Close()

	items := make([]RegistrationRequestResponse, 0)
	for rows.Next() {
		var item RegistrationRequestResponse
		var reviewedAt sql.NullTime
		if err := rows.Scan(
			&item.ID,
			&item.FirstName,
			&item.LastName,
			&item.MiddleName,
			&item.FullName,
			&item.Username,
			&item.Email,
			&item.Position,
			&item.Role,
			&item.Status,
			&item.RejectionReason,
			&item.CreatedAt,
			&reviewedAt,
			&item.ReviewedByName,
		); err != nil {
			c.JSON(500, errorResponse{Error: "failed to parse registration requests"})
			return
		}
		if reviewedAt.Valid {
			value := reviewedAt.Time
			item.ReviewedAt = &value
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to iterate registration requests"})
		return
	}

	totalPages := total / limit
	if total%limit != 0 {
		totalPages++
	}

	c.JSON(200, listRegistrationRequestsResponse{
		Items: items,
		Meta: listMeta{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

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

func (h *Handler) createUser(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	firstName := strings.TrimSpace(req.FirstName)
	lastName := strings.TrimSpace(req.LastName)
	middleName := strings.TrimSpace(req.MiddleName)
	position := strings.TrimSpace(req.Position)
	email, err := validateEmailAddress(req.Email)
	if err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}
	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)
	confirmPassword := strings.TrimSpace(req.ConfirmPassword)
	role := normalizeRole(req.Role)

	if firstName == "" || lastName == "" || position == "" || username == "" {
		c.JSON(400, errorResponse{Error: "first_name, last_name, position, email and username are required"})
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

	if err := h.assertUserIdentityAvailable(username, email); err != nil {
		switch {
		case errors.Is(err, errUserExists):
			c.JSON(409, errorResponse{Error: "username already exists"})
		case errors.Is(err, errEmailExists):
			c.JSON(409, errorResponse{Error: "email already exists"})
		default:
			c.JSON(500, errorResponse{Error: "failed to validate identity"})
		}
		return
	}

	passwordHash, err := hashPassword(password)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to secure password"})
		return
	}

	fullName := composeFullName(lastName, firstName, middleName)

	var created UserResponse
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
		RETURNING id, first_name, last_name, middle_name, full_name, username, COALESCE(email, ''), COALESCE(position, ''), role, created_at
	`, firstName, lastName, middleName, fullName, username, email, position, passwordHash, role).Scan(
		&created.ID,
		&created.FirstName,
		&created.LastName,
		&created.MiddleName,
		&created.FullName,
		&created.Username,
		&created.Email,
		&created.Position,
		&created.Role,
		&created.CreatedAt,
	)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to create user"})
		return
	}

	c.JSON(201, created)
}

func (h *Handler) deleteUser(c *gin.Context) {
	userID, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(400, errorResponse{Error: "invalid user id"})
		return
	}

	currentUser := middleware.CurrentUser(c)
	if currentUser == nil {
		c.JSON(401, errorResponse{Error: "unauthorized"})
		return
	}

	if currentUser.ID == userID {
		c.JSON(400, errorResponse{Error: "cannot delete current user"})
		return
	}

	var role string
	err = h.db.QueryRow(`
		SELECT role
		FROM users
		WHERE id = $1
		LIMIT 1
	`, userID).Scan(&role)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(404, errorResponse{Error: "user not found"})
			return
		}
		c.JSON(500, errorResponse{Error: "failed to load user"})
		return
	}

	if role == "admin" {
		var adminCount int
		if err := h.db.QueryRow(`
			SELECT COUNT(*)
			FROM users
			WHERE role = 'admin'
		`).Scan(&adminCount); err != nil {
			c.JSON(500, errorResponse{Error: "failed to validate admin pool"})
			return
		}

		if adminCount <= 1 {
			c.JSON(400, errorResponse{Error: "cannot delete the last admin"})
			return
		}
	}

	tx, err := h.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to start delete transaction"})
		return
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
		UPDATE plan_indicator_details pid
		SET responsible_user_ids = COALESCE((
			SELECT jsonb_agg(elem.value)
			FROM jsonb_array_elements_text(COALESCE(pid.responsible_user_ids, '[]'::jsonb)) AS elem(value)
			WHERE elem.value <> $1::TEXT
		), '[]'::jsonb),
		    updated_at = NOW()
		WHERE COALESCE(pid.responsible_user_ids, '[]'::jsonb) <> '[]'::jsonb
	`, strconv.FormatInt(userID, 10)); err != nil {
		c.JSON(500, errorResponse{Error: "failed to clear legacy responsibilities"})
		return
	}

	result, err := tx.Exec(`
		DELETE FROM users
		WHERE id = $1
	`, userID)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to delete user"})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to finalize delete"})
		return
	}
	if rowsAffected == 0 {
		c.JSON(404, errorResponse{Error: "user not found"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to commit delete"})
		return
	}

	c.JSON(200, gin.H{"message": "user deleted"})
}

func (h *Handler) approveRegistrationRequest(c *gin.Context) {
	requestID, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || requestID <= 0 {
		c.JSON(400, errorResponse{Error: "invalid request id"})
		return
	}

	currentUser := middleware.CurrentUser(c)
	if currentUser == nil {
		c.JSON(401, errorResponse{Error: "unauthorized"})
		return
	}

	var req RegistrationRequestResponse
	var passwordHash string
	err = h.db.QueryRow(`
		SELECT id,
		       first_name,
		       last_name,
		       middle_name,
		       full_name,
		       username,
		       email,
		       COALESCE(position, ''),
		       role,
		       status,
		       password_hash
		FROM registration_requests
		WHERE id = $1
		LIMIT 1
	`, requestID).Scan(
		&req.ID,
		&req.FirstName,
		&req.LastName,
		&req.MiddleName,
		&req.FullName,
		&req.Username,
		&req.Email,
		&req.Position,
		&req.Role,
		&req.Status,
		&passwordHash,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(404, errorResponse{Error: "registration request not found"})
			return
		}
		c.JSON(500, errorResponse{Error: "failed to load registration request"})
		return
	}

	if req.Status != "pending" {
		c.JSON(400, errorResponse{Error: "only pending requests can be approved"})
		return
	}

	if err := h.assertUserIdentityAvailable(req.Username, req.Email); err != nil {
		switch {
		case errors.Is(err, errUserExists):
			c.JSON(409, errorResponse{Error: "username already exists"})
		case errors.Is(err, errEmailExists):
			c.JSON(409, errorResponse{Error: "email already exists"})
		default:
			c.JSON(500, errorResponse{Error: "failed to validate identity"})
		}
		return
	}

	var approvedUserID int64
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
		RETURNING id
	`, req.FirstName, req.LastName, req.MiddleName, req.FullName, req.Username, req.Email, req.Position, passwordHash, req.Role).Scan(&approvedUserID)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to create approved user"})
		return
	}

	if _, err := h.db.Exec(`
		UPDATE registration_requests
		SET status = 'approved',
		    rejection_reason = '',
		    reviewed_by = $1,
		    reviewed_at = NOW(),
		    approved_user_id = $2,
		    updated_at = NOW()
		WHERE id = $3
	`, currentUser.ID, approvedUserID, requestID); err != nil {
		c.JSON(500, errorResponse{Error: "failed to finalize registration request"})
		return
	}

	c.JSON(200, gin.H{"message": "registration request approved"})
}

func (h *Handler) rejectRegistrationRequest(c *gin.Context) {
	requestID, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || requestID <= 0 {
		c.JSON(400, errorResponse{Error: "invalid request id"})
		return
	}

	currentUser := middleware.CurrentUser(c)
	if currentUser == nil {
		c.JSON(401, errorResponse{Error: "unauthorized"})
		return
	}

	var body rejectRegistrationRequestBody
	_ = c.ShouldBindJSON(&body)

	var status string
	err = h.db.QueryRow(`
		SELECT status
		FROM registration_requests
		WHERE id = $1
		LIMIT 1
	`, requestID).Scan(&status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(404, errorResponse{Error: "registration request not found"})
			return
		}
		c.JSON(500, errorResponse{Error: "failed to load registration request"})
		return
	}

	if status != "pending" {
		c.JSON(400, errorResponse{Error: "only pending requests can be rejected"})
		return
	}

	if _, err := h.db.Exec(`
		UPDATE registration_requests
		SET status = 'rejected',
		    rejection_reason = $1,
		    reviewed_by = $2,
		    reviewed_at = NOW(),
		    updated_at = NOW()
		WHERE id = $3
	`, strings.TrimSpace(body.Reason), currentUser.ID, requestID); err != nil {
		c.JSON(500, errorResponse{Error: "failed to reject registration request"})
		return
	}

	c.JSON(200, gin.H{"message": "registration request rejected"})
}

var (
	errUserExists  = errors.New("username already exists")
	errEmailExists = errors.New("email already exists")
)

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

func hashPassword(password string) (string, error) {
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashBytes), nil
}

func (h *Handler) assertUserIdentityAvailable(username, email string) error {
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

	var id int64
	err := h.db.QueryRow(query, args...).Scan(&id)
	if err == nil {
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
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	return nil
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

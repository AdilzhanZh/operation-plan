package task

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"OperationPlan/internal/middleware"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	db *sql.DB
}

type Task struct {
	ID                int        `json:"id"`
	PlanID            int        `json:"plan_id"`
	ParentID          *int       `json:"parent_id,omitempty"`
	Title             string     `json:"title"`
	Description       string     `json:"description"`
	PlannedValue      string     `json:"planned_value"`
	Deadline          *time.Time `json:"deadline,omitempty"`
	ResponsibleUserID int        `json:"responsible_user_id"`
	Status            string     `json:"status"`
	ResultText        string     `json:"result_text"`
	CompletionPercent int        `json:"completion_percent"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type createTaskRequest struct {
	PlanID            int    `json:"plan_id" binding:"required"`
	ParentID          *int   `json:"parent_id"`
	Title             string `json:"title" binding:"required"`
	Description       string `json:"description"`
	PlannedValue      string `json:"planned_value"`
	Deadline          string `json:"deadline" binding:"required"`
	ResponsibleUserID int    `json:"responsible_user_id" binding:"required"`
}

type updateTaskRequest struct {
	Title             *string `json:"title"`
	Description       *string `json:"description"`
	PlannedValue      *string `json:"planned_value"`
	Deadline          *string `json:"deadline"`
	ResponsibleUserID *int    `json:"responsible_user_id"`
	ResultText        *string `json:"result_text"`
	CompletionPercent *int    `json:"completion_percent"`
}

type updateTaskStatusRequest struct {
	Status  string `json:"status" binding:"required"`
	Comment string `json:"comment"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type taskListResponse struct {
	Items []Task `json:"items"`
}

type updateTaskStatusResponse struct {
	Task    Task   `json:"task"`
	Comment string `json:"comment"`
}

type rowScanner interface {
	Scan(dest ...any) error
}

func RegisterRoutes(router gin.IRouter, db *sql.DB) {
	h := &Handler{db: db}

	router.GET("/tasks", h.listTasks)
	router.GET("/tasks/:id", h.getTask)

	adminOnly := router.Group("/")
	adminOnly.Use(middleware.RequireRoles("admin"))
	adminOnly.POST("/tasks", h.createTask)
	adminOnly.PATCH("/tasks/:id", h.updateTask)
	adminOnly.PATCH("/tasks/:id/status", h.updateTaskStatus)
	adminOnly.DELETE("/tasks/:id", h.deleteTask)
}

// listTasks godoc
// @Summary List tasks
// @Description Returns tasks with current statuses
// @Tags tasks
// @Produce json
// @Security BearerAuth
// @Success 200 {object} taskListResponse
// @Router /tasks [get]
func (h *Handler) listTasks(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(401, errorResponse{Error: "unauthorized"})
		return
	}

	query := `
		SELECT id, plan_id, parent_id, title, description, planned_value, deadline,
		       responsible_user_id, status, result_text, completion_percent, created_at, updated_at
		FROM tasks
	`

	var (
		rows *sql.Rows
		err  error
	)
	if user.Role == "admin" {
		rows, err = h.db.Query(query + ` ORDER BY id ASC`)
	} else {
		rows, err = h.db.Query(query+`
			WHERE responsible_user_id = $1
			ORDER BY id ASC
		`, user.ID)
	}
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to load tasks"})
		return
	}
	defer rows.Close()

	items := make([]Task, 0)
	for rows.Next() {
		task, scanErr := scanTask(rows)
		if scanErr != nil {
			c.JSON(500, errorResponse{Error: "failed to parse tasks"})
			return
		}
		items = append(items, task)
	}

	if err := rows.Err(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to iterate tasks"})
		return
	}

	c.JSON(200, taskListResponse{Items: items})
}

// createTask godoc
// @Summary Create task
// @Description Creates a new task within selected plan
// @Tags tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param payload body createTaskRequest true "Task payload"
// @Success 201 {object} Task
// @Failure 400 {object} errorResponse
// @Router /tasks [post]
func (h *Handler) createTask(c *gin.Context) {
	var req createTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	deadline, err := time.Parse("2006-01-02", req.Deadline)
	if err != nil {
		c.JSON(400, errorResponse{Error: "deadline must be YYYY-MM-DD"})
		return
	}

	parentID := any(nil)
	if req.ParentID != nil {
		parentID = *req.ParentID
	}

	row := h.db.QueryRow(`
		INSERT INTO tasks (
			plan_id, parent_id, title, description, planned_value, deadline,
			responsible_user_id, status, result_text, completion_percent, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'created', '', 0, NOW(), NOW())
		RETURNING id, plan_id, parent_id, title, description, planned_value, deadline,
		          responsible_user_id, status, result_text, completion_percent, created_at, updated_at
	`, req.PlanID, parentID, req.Title, req.Description, req.PlannedValue, deadline, req.ResponsibleUserID)

	task, scanErr := scanTask(row)
	if scanErr != nil {
		c.JSON(500, errorResponse{Error: "failed to create task"})
		return
	}

	c.JSON(201, task)
}

// getTask godoc
// @Summary Get task
// @Description Returns task by ID
// @Tags tasks
// @Produce json
// @Security BearerAuth
// @Param id path int true "Task ID"
// @Success 200 {object} Task
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /tasks/{id} [get]
func (h *Handler) getTask(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(401, errorResponse{Error: "unauthorized"})
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, errorResponse{Error: "invalid task id"})
		return
	}

	task, err := h.fetchTaskByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(404, errorResponse{Error: "task not found"})
			return
		}

		c.JSON(500, errorResponse{Error: "failed to load task"})
		return
	}

	if !canAccessTask(user, task) {
		c.JSON(403, errorResponse{Error: "forbidden"})
		return
	}

	c.JSON(200, task)
}

// updateTask godoc
// @Summary Update task
// @Description Updates task fields
// @Tags tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Task ID"
// @Param payload body updateTaskRequest true "Task patch payload"
// @Success 200 {object} Task
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /tasks/{id} [patch]
func (h *Handler) updateTask(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, errorResponse{Error: "invalid task id"})
		return
	}

	var req updateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	existing, err := h.fetchTaskByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(404, errorResponse{Error: "task not found"})
			return
		}

		c.JSON(500, errorResponse{Error: "failed to load task"})
		return
	}

	if req.Title != nil {
		existing.Title = strings.TrimSpace(*req.Title)
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.PlannedValue != nil {
		existing.PlannedValue = *req.PlannedValue
	}
	if req.Deadline != nil {
		trimmed := strings.TrimSpace(*req.Deadline)
		if trimmed == "" {
			existing.Deadline = nil
		} else {
			deadline, parseErr := time.Parse("2006-01-02", trimmed)
			if parseErr != nil {
				c.JSON(400, errorResponse{Error: "deadline must be YYYY-MM-DD"})
				return
			}
			existing.Deadline = &deadline
		}
	}
	if req.ResponsibleUserID != nil {
		existing.ResponsibleUserID = *req.ResponsibleUserID
	}
	if req.ResultText != nil {
		existing.ResultText = *req.ResultText
	}
	if req.CompletionPercent != nil {
		existing.CompletionPercent = *req.CompletionPercent
	}

	deadlineArg := any(nil)
	if existing.Deadline != nil {
		deadlineArg = *existing.Deadline
	}

	row := h.db.QueryRow(`
		UPDATE tasks
		SET title = $1,
		    description = $2,
		    planned_value = $3,
		    deadline = $4,
		    responsible_user_id = $5,
		    result_text = $6,
		    completion_percent = $7,
		    updated_at = NOW()
		WHERE id = $8
		RETURNING id, plan_id, parent_id, title, description, planned_value, deadline,
		          responsible_user_id, status, result_text, completion_percent, created_at, updated_at
	`, existing.Title, existing.Description, existing.PlannedValue, deadlineArg, existing.ResponsibleUserID, existing.ResultText, existing.CompletionPercent, id)

	updated, scanErr := scanTask(row)
	if scanErr != nil {
		if errors.Is(scanErr, sql.ErrNoRows) {
			c.JSON(404, errorResponse{Error: "task not found"})
			return
		}

		c.JSON(500, errorResponse{Error: "failed to update task"})
		return
	}

	c.JSON(200, updated)
}

// updateTaskStatus godoc
// @Summary Update task status
// @Description Changes lifecycle status of selected task
// @Tags tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Task ID"
// @Param payload body updateTaskStatusRequest true "Status payload"
// @Success 200 {object} updateTaskStatusResponse
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /tasks/{id}/status [patch]
func (h *Handler) updateTaskStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, errorResponse{Error: "invalid task id"})
		return
	}

	var req updateTaskStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	row := h.db.QueryRow(`
		UPDATE tasks
		SET status = $1,
		    updated_at = NOW()
		WHERE id = $2
		RETURNING id, plan_id, parent_id, title, description, planned_value, deadline,
		          responsible_user_id, status, result_text, completion_percent, created_at, updated_at
	`, req.Status, id)

	updated, scanErr := scanTask(row)
	if scanErr != nil {
		if errors.Is(scanErr, sql.ErrNoRows) {
			c.JSON(404, errorResponse{Error: "task not found"})
			return
		}

		c.JSON(500, errorResponse{Error: "failed to update task status"})
		return
	}

	c.JSON(200, updateTaskStatusResponse{Task: updated, Comment: req.Comment})
}

// deleteTask godoc
// @Summary Delete task
// @Description Deletes task by ID
// @Tags tasks
// @Produce json
// @Security BearerAuth
// @Param id path int true "Task ID"
// @Success 204
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /tasks/{id} [delete]
func (h *Handler) deleteTask(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, errorResponse{Error: "invalid task id"})
		return
	}

	result, err := h.db.Exec(`DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to delete task"})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to check delete result"})
		return
	}

	if rowsAffected == 0 {
		c.JSON(404, errorResponse{Error: "task not found"})
		return
	}

	c.Status(204)
}

func (h *Handler) fetchTaskByID(id int) (Task, error) {
	row := h.db.QueryRow(`
		SELECT id, plan_id, parent_id, title, description, planned_value, deadline,
		       responsible_user_id, status, result_text, completion_percent, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`, id)

	return scanTask(row)
}

func scanTask(scanner rowScanner) (Task, error) {
	var task Task
	var parentID sql.NullInt64
	var deadline sql.NullTime
	var resultText sql.NullString
	var completionPercent sql.NullInt64

	err := scanner.Scan(
		&task.ID,
		&task.PlanID,
		&parentID,
		&task.Title,
		&task.Description,
		&task.PlannedValue,
		&deadline,
		&task.ResponsibleUserID,
		&task.Status,
		&resultText,
		&completionPercent,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	if err != nil {
		return Task{}, err
	}

	if parentID.Valid {
		value := int(parentID.Int64)
		task.ParentID = &value
	}

	if deadline.Valid {
		value := deadline.Time
		task.Deadline = &value
	}

	if resultText.Valid {
		task.ResultText = resultText.String
	}

	if completionPercent.Valid {
		task.CompletionPercent = int(completionPercent.Int64)
	}

	return task, nil
}

func canAccessTask(user *middleware.UserContext, task Task) bool {
	if user == nil {
		return false
	}

	if user.Role == "admin" {
		return true
	}

	return int64(task.ResponsibleUserID) == user.ID
}

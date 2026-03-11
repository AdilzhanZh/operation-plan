package report

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"OperationPlan/internal/files"
	"OperationPlan/internal/middleware"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	db *sql.DB
}

type Report struct {
	ID         int       `json:"id"`
	TaskID     int       `json:"task_id"`
	FilePath   string    `json:"file_path"`
	UploadedBy int       `json:"uploaded_by"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type reportListResponse struct {
	Items []Report `json:"items"`
}

func RegisterRoutes(router gin.IRouter, db *sql.DB) {
	h := &Handler{db: db}

	router.POST("/tasks/:id/report", h.uploadReport)
	router.GET("/tasks/:id/reports", h.listReports)
}

// uploadReport godoc
// @Summary Upload report file
// @Description Uploads supporting document for task
// @Tags reports
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param id path int true "Task ID"
// @Param file formData file true "Report file (pdf/docx/xlsx/jpg/png)"
// @Success 201 {object} Report
// @Failure 400 {object} errorResponse
// @Router /tasks/{id}/report [post]
func (h *Handler) uploadReport(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(401, errorResponse{Error: "unauthorized"})
		return
	}

	taskID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, errorResponse{Error: "invalid task id"})
		return
	}

	responsibleUserID, err := h.fetchTaskResponsibleUserID(taskID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(404, errorResponse{Error: "task not found"})
			return
		}
		c.JSON(500, errorResponse{Error: "failed to load task"})
		return
	}
	if !canAccessTaskReports(user, responsibleUserID) {
		c.JSON(403, errorResponse{Error: "forbidden"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, errorResponse{Error: "file is required"})
		return
	}

	if err := files.ValidateUpload(fileHeader); err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	reportDir, err := resolveTaskReportStoragePath(taskID)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to prepare report storage"})
		return
	}
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		c.JSON(500, errorResponse{Error: "failed to prepare report storage"})
		return
	}

	storedName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(fileHeader.Filename))
	filePath := filepath.Join(reportDir, storedName)
	if err := c.SaveUploadedFile(fileHeader, filePath); err != nil {
		c.JSON(500, errorResponse{Error: "failed to store report file"})
		return
	}

	var report Report
	err = h.db.QueryRow(`
		INSERT INTO reports (task_id, file_path, uploaded_by, uploaded_at)
		VALUES ($1, $2, $3, NOW())
		RETURNING id, task_id, file_path, uploaded_by, uploaded_at
	`, taskID, filePath, user.ID).Scan(
		&report.ID,
		&report.TaskID,
		&report.FilePath,
		&report.UploadedBy,
		&report.UploadedAt,
	)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to create report"})
		return
	}

	c.JSON(201, report)
}

// listReports godoc
// @Summary List task reports
// @Description Returns uploaded reports for selected task
// @Tags reports
// @Produce json
// @Security BearerAuth
// @Param id path int true "Task ID"
// @Success 200 {object} reportListResponse
// @Failure 400 {object} errorResponse
// @Router /tasks/{id}/reports [get]
func (h *Handler) listReports(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(401, errorResponse{Error: "unauthorized"})
		return
	}

	taskID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, errorResponse{Error: "invalid task id"})
		return
	}

	responsibleUserID, err := h.fetchTaskResponsibleUserID(taskID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(404, errorResponse{Error: "task not found"})
			return
		}
		c.JSON(500, errorResponse{Error: "failed to load task"})
		return
	}
	if !canAccessTaskReports(user, responsibleUserID) {
		c.JSON(403, errorResponse{Error: "forbidden"})
		return
	}

	rows, err := h.db.Query(`
		SELECT id, task_id, file_path, uploaded_by, uploaded_at
		FROM reports
		WHERE task_id = $1
		ORDER BY id ASC
	`, taskID)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to load reports"})
		return
	}
	defer rows.Close()

	items := make([]Report, 0)
	for rows.Next() {
		var report Report
		if err := rows.Scan(&report.ID, &report.TaskID, &report.FilePath, &report.UploadedBy, &report.UploadedAt); err != nil {
			c.JSON(500, errorResponse{Error: "failed to parse reports"})
			return
		}
		items = append(items, report)
	}

	if err := rows.Err(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to iterate reports"})
		return
	}

	c.JSON(200, reportListResponse{Items: items})
}

func (h *Handler) fetchTaskResponsibleUserID(taskID int) (int64, error) {
	var responsibleUserID int64
	err := h.db.QueryRow(`
		SELECT responsible_user_id
		FROM tasks
		WHERE id = $1
	`, taskID).Scan(&responsibleUserID)
	if err != nil {
		return 0, err
	}

	return responsibleUserID, nil
}

func canAccessTaskReports(user *middleware.UserContext, responsibleUserID int64) bool {
	if user == nil {
		return false
	}

	if user.Role == "admin" {
		return true
	}

	return user.ID == responsibleUserID
}

func resolveTaskReportStoragePath(taskID int) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return filepath.Join(wd, "storage", "task-reports", strconv.Itoa(taskID)), nil
}

package report

import (
	"database/sql"
	"fmt"
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

func RegisterRoutes(router gin.IRoutes, db *sql.DB) {
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
	taskID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, errorResponse{Error: "invalid task id"})
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

	uploadedBy := 1
	if user := middleware.CurrentUser(c); user != nil {
		uploadedBy = int(user.ID)
	}

	filePath := fmt.Sprintf("/uploads/tasks/%d/%s", taskID, filepath.Base(fileHeader.Filename))

	var report Report
	err = h.db.QueryRow(`
		INSERT INTO reports (task_id, file_path, uploaded_by, uploaded_at)
		VALUES ($1, $2, $3, NOW())
		RETURNING id, task_id, file_path, uploaded_by, uploaded_at
	`, taskID, filePath, uploadedBy).Scan(
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
	taskID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, errorResponse{Error: "invalid task id"})
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

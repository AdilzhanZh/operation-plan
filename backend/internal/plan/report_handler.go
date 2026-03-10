package plan

import (
	"database/sql"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"OperationPlan/internal/files"
	"OperationPlan/internal/middleware"

	"github.com/gin-gonic/gin"
)

const (
	reportStatusPending  = "pending"
	reportStatusApproved = "approved"
	reportStatusRejected = "rejected"
)

var errReportAlreadySubmitted = errors.New("report already submitted")

type planIndicatorReportFile struct {
	ID          int    `json:"id"`
	FileName    string `json:"file_name"`
	DownloadURL string `json:"download_url"`
}

type planIndicatorReportRow struct {
	ID                   int                       `json:"id"`
	IndicatorID          int                       `json:"indicator_id"`
	Year                 int                       `json:"year"`
	DevelopmentIndicator string                    `json:"development_indicator"`
	PlannedValue         string                    `json:"planned_value"`
	Unit                 string                    `json:"unit"`
	ExecutionDeadline    string                    `json:"execution_deadline"`
	Responsible          string                    `json:"responsible"`
	ReportText           string                    `json:"report_text"`
	Status               string                    `json:"status"`
	ReviewNote           string                    `json:"review_note"`
	ApprovalFormula      string                    `json:"approval_formula"`
	SubmittedBy          int64                     `json:"submitted_by"`
	SubmittedByName      string                    `json:"submitted_by_name"`
	SubmittedAt          time.Time                 `json:"submitted_at"`
	ReviewedByName       string                    `json:"reviewed_by_name"`
	ReviewedAt           time.Time                 `json:"reviewed_at,omitempty"`
	Files                []planIndicatorReportFile `json:"files"`
}

type planIndicatorReportListResponse struct {
	Year  int                      `json:"year"`
	Items []planIndicatorReportRow `json:"items"`
}

type reviewPlanIndicatorReportRequest struct {
	Action          string `json:"action" binding:"required"`
	ReviewNote      string `json:"review_note"`
	ApprovalFormula string `json:"approval_formula"`
}

// submitPlanIndicatorReport godoc
// @Summary Submit indicator report
// @Description Prorector submits report text and one or many files for assigned indicator. At least one file is required.
// @Tags plans
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param indicator_id path int true "Planning period indicator ID"
// @Param year query int true "Year"
// @Param report_text formData string false "Report text"
// @Param files formData file true "Supporting files"
// @Success 201 {object} planIndicatorReportRow
// @Failure 400 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Failure 409 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /plans/indicators/{indicator_id}/report [post]
func (h *Handler) submitPlanIndicatorReport(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(401, errorResponse{Error: "unauthorized"})
		return
	}
	if user.Role != "prorector" {
		c.JSON(403, errorResponse{Error: "forbidden"})
		return
	}

	indicatorID, err := strconv.Atoi(c.Param("indicator_id"))
	if err != nil {
		c.JSON(400, errorResponse{Error: "invalid indicator id"})
		return
	}

	year, yearKey, err := parseYear(c.Query("year"))
	if err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	if err := h.ensurePlanningPeriodTable(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to prepare planning period storage"})
		return
	}
	if err := h.ensurePlanIndicatorDetailsTable(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to prepare plans storage"})
		return
	}
	if err := h.ensurePlanIndicatorReportsTable(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to prepare reports storage"})
		return
	}
	if err := h.ensurePlanIndicatorReportFilesTable(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to prepare report files storage"})
		return
	}

	if err := h.ensureIndicatorYearExists(indicatorID, yearKey); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(404, errorResponse{Error: "indicator for selected year not found"})
			return
		}
		c.JSON(500, errorResponse{Error: "failed to validate indicator"})
		return
	}

	var responsibleUserIDsRaw []byte
	err = h.db.QueryRow(`
		SELECT COALESCE(responsible_user_ids, '[]'::jsonb)
		FROM plan_indicator_details
		WHERE planning_period_indicator_id = $1
		  AND year = $2
		LIMIT 1
	`, indicatorID, year).Scan(&responsibleUserIDsRaw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(403, errorResponse{Error: "indicator is not assigned to prorector"})
			return
		}
		c.JSON(500, errorResponse{Error: "failed to validate responsibility"})
		return
	}

	responsibleUserIDs, parseErr := parseResponsibleUserIDs(responsibleUserIDsRaw)
	if parseErr != nil {
		c.JSON(500, errorResponse{Error: "failed to validate responsibility"})
		return
	}
	if !containsInt64(responsibleUserIDs, user.ID) {
		c.JSON(403, errorResponse{Error: "indicator is not assigned to this prorector"})
		return
	}

	reportText := strings.TrimSpace(c.PostForm("report_text"))
	uploadedFiles, err := extractReportFiles(c)
	if err != nil {
		c.JSON(400, errorResponse{Error: "invalid files payload"})
		return
	}
	if len(uploadedFiles) == 0 {
		c.JSON(400, errorResponse{Error: "at least one file is required"})
		return
	}
	for _, fileHeader := range uploadedFiles {
		if err := files.ValidateUpload(fileHeader); err != nil {
			c.JSON(400, errorResponse{Error: err.Error()})
			return
		}
	}

	item, err := h.saveIndicatorReport(c, indicatorID, year, reportText, uploadedFiles, user.ID)
	if err != nil {
		if errors.Is(err, errReportAlreadySubmitted) {
			c.JSON(409, errorResponse{Error: "report already submitted and waiting for review"})
			return
		}
		c.JSON(500, errorResponse{Error: "failed to save report"})
		return
	}

	c.JSON(201, item)
}

// listPlanReports godoc
// @Summary List submitted indicator reports
// @Description Returns indicator reports for selected year
// @Tags plans
// @Produce json
// @Security BearerAuth
// @Param year query int true "Year"
// @Param status query string false "Optional statuses: pending,approved,rejected. Comma-separated."
// @Success 200 {object} planIndicatorReportListResponse
// @Failure 400 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /plans/reports [get]
func (h *Handler) listPlanReports(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(401, errorResponse{Error: "unauthorized"})
		return
	}
	if user.Role != "admin" && user.Role != "prorector" {
		c.JSON(403, errorResponse{Error: "forbidden"})
		return
	}

	year, _, err := parseYear(c.Query("year"))
	if err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}
	statuses, err := parseReportStatuses(c.Query("status"))
	if err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	if err := h.ensurePlanningPeriodTable(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to prepare planning period storage"})
		return
	}
	if err := h.ensurePlanIndicatorDetailsTable(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to prepare plans storage"})
		return
	}
	if err := h.ensurePlanIndicatorReportsTable(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to prepare reports storage"})
		return
	}
	if err := h.ensurePlanIndicatorReportFilesTable(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to prepare report files storage"})
		return
	}

	args := make([]any, 0, 4+len(statuses))
	where := make([]string, 0, 4)
	args = append(args, year)
	where = append(where, "pir.year = $1")

	if user.Role == "prorector" {
		args = append(args, user.ID)
		where = append(where, fmt.Sprintf("pir.submitted_by = $%d", len(args)))
	}

	if len(statuses) > 0 {
		placeholders := make([]string, 0, len(statuses))
		for _, status := range statuses {
			args = append(args, status)
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)))
		}
		where = append(where, fmt.Sprintf("pir.status IN (%s)", strings.Join(placeholders, ", ")))
	}

	items, err := h.queryPlanReports(strings.Join(where, " AND "), args...)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to load reports"})
		return
	}

	c.JSON(200, planIndicatorReportListResponse{
		Year:  year,
		Items: items,
	})
}

// reviewPlanIndicatorReport godoc
// @Summary Review submitted indicator report
// @Description Admin approves or rejects report
// @Tags plans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param report_id path int true "Report ID"
// @Param payload body reviewPlanIndicatorReportRequest true "Review payload"
// @Success 200 {object} planIndicatorReportRow
// @Failure 400 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /plans/reports/{report_id}/review [patch]
func (h *Handler) reviewPlanIndicatorReport(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(401, errorResponse{Error: "unauthorized"})
		return
	}
	if user.Role != "admin" {
		c.JSON(403, errorResponse{Error: "forbidden"})
		return
	}

	reportID, err := strconv.Atoi(c.Param("report_id"))
	if err != nil {
		c.JSON(400, errorResponse{Error: "invalid report id"})
		return
	}

	var req reviewPlanIndicatorReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	if err := h.ensurePlanIndicatorReportsTable(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to prepare reports storage"})
		return
	}
	if err := h.ensurePlanIndicatorReportFilesTable(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to prepare report files storage"})
		return
	}

	action := strings.ToLower(strings.TrimSpace(req.Action))

	switch action {
	case "approve":
		formula := strings.TrimSpace(req.ApprovalFormula)
		if formula == "" {
			c.JSON(400, errorResponse{Error: "approval_formula is required"})
			return
		}

		err = h.db.QueryRow(`
			UPDATE plan_indicator_reports
			SET status = $1,
			    review_note = '',
			    approval_formula = $2,
			    reviewed_by = $3,
			    reviewed_at = NOW(),
			    updated_at = NOW()
			WHERE id = $4
			RETURNING id
		`, reportStatusApproved, formula, user.ID, reportID).Scan(&reportID)
	case "reject":
		reviewNote := strings.TrimSpace(req.ReviewNote)
		if reviewNote == "" {
			c.JSON(400, errorResponse{Error: "review_note is required"})
			return
		}

		err = h.db.QueryRow(`
			UPDATE plan_indicator_reports
			SET status = $1,
			    review_note = $2,
			    approval_formula = '',
			    reviewed_by = $3,
			    reviewed_at = NOW(),
			    updated_at = NOW()
			WHERE id = $4
			RETURNING id
		`, reportStatusRejected, reviewNote, user.ID, reportID).Scan(&reportID)
	default:
		c.JSON(400, errorResponse{Error: "action must be approve or reject"})
		return
	}

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(404, errorResponse{Error: "report not found"})
			return
		}
		c.JSON(500, errorResponse{Error: "failed to review report"})
		return
	}

	item, err := h.fetchPlanReportByID(reportID)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to load reviewed report"})
		return
	}

	c.JSON(200, item)
}

// downloadPlanReportFile godoc
// @Summary Download report file
// @Description Downloads attached file from submitted indicator report
// @Tags plans
// @Produce octet-stream
// @Security BearerAuth
// @Param file_id path int true "Report file ID"
// @Success 200 {file} file
// @Failure 400 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /plans/reports/files/{file_id}/download [get]
func (h *Handler) downloadPlanReportFile(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(401, errorResponse{Error: "unauthorized"})
		return
	}
	if user.Role != "admin" && user.Role != "prorector" {
		c.JSON(403, errorResponse{Error: "forbidden"})
		return
	}

	fileID, err := strconv.Atoi(c.Param("file_id"))
	if err != nil {
		c.JSON(400, errorResponse{Error: "invalid file id"})
		return
	}

	if err := h.ensurePlanIndicatorReportsTable(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to prepare reports storage"})
		return
	}
	if err := h.ensurePlanIndicatorReportFilesTable(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to prepare report files storage"})
		return
	}

	var fileName string
	var storagePath string
	var submittedBy int64
	err = h.db.QueryRow(`
		SELECT rf.file_name,
		       rf.storage_path,
		       rr.submitted_by
		FROM plan_indicator_report_files rf
		JOIN plan_indicator_reports rr
		  ON rr.id = rf.report_id
		WHERE rf.id = $1
		LIMIT 1
	`, fileID).Scan(&fileName, &storagePath, &submittedBy)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(404, errorResponse{Error: "file not found"})
			return
		}
		c.JSON(500, errorResponse{Error: "failed to load file"})
		return
	}

	if user.Role == "prorector" && submittedBy != user.ID {
		c.JSON(403, errorResponse{Error: "forbidden"})
		return
	}

	resolvedPath := strings.TrimSpace(storagePath)
	if resolvedPath == "" {
		c.JSON(404, errorResponse{Error: "file is missing"})
		return
	}

	finalPath, err := resolveExistingReportFilePath(resolvedPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			c.JSON(404, errorResponse{Error: "file is missing on server. Please ask prorector to resubmit report with files"})
			return
		}
		c.JSON(500, errorResponse{Error: "failed to access file"})
		return
	}

	downloadName := strings.TrimSpace(fileName)
	if downloadName == "" {
		downloadName = filepath.Base(finalPath)
	}
	c.FileAttachment(finalPath, downloadName)
}

func (h *Handler) ensurePlanIndicatorReportsTable() error {
	_, err := h.db.Exec(`
		CREATE TABLE IF NOT EXISTS plan_indicator_reports (
			id BIGSERIAL PRIMARY KEY,
			planning_period_indicator_id BIGINT NOT NULL REFERENCES planning_period_indicators(id) ON DELETE CASCADE,
			year INT NOT NULL,
			report_text TEXT NOT NULL DEFAULT '',
			file_path VARCHAR(1024) NOT NULL DEFAULT '',
			file_name VARCHAR(255) NOT NULL DEFAULT '',
			status VARCHAR(32) NOT NULL DEFAULT 'pending',
			review_note TEXT NOT NULL DEFAULT '',
			approval_formula TEXT NOT NULL DEFAULT '',
			reviewed_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
			reviewed_at TIMESTAMPTZ NULL,
			submitted_by BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (planning_period_indicator_id, year, submitted_by)
		);
	`)
	if err != nil {
		return err
	}

	_, err = h.db.Exec(`
		ALTER TABLE plan_indicator_reports
		ADD COLUMN IF NOT EXISTS file_name VARCHAR(255) NOT NULL DEFAULT '';
	`)
	if err != nil {
		return err
	}

	_, err = h.db.Exec(`
		ALTER TABLE plan_indicator_reports
		ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
	`)
	if err != nil {
		return err
	}

	_, err = h.db.Exec(`
		ALTER TABLE plan_indicator_reports
		ADD COLUMN IF NOT EXISTS status VARCHAR(32) NOT NULL DEFAULT 'pending';
	`)
	if err != nil {
		return err
	}

	_, err = h.db.Exec(`
		ALTER TABLE plan_indicator_reports
		ADD COLUMN IF NOT EXISTS review_note TEXT NOT NULL DEFAULT '';
	`)
	if err != nil {
		return err
	}

	_, err = h.db.Exec(`
		ALTER TABLE plan_indicator_reports
		ADD COLUMN IF NOT EXISTS approval_formula TEXT NOT NULL DEFAULT '';
	`)
	if err != nil {
		return err
	}

	_, err = h.db.Exec(`
		ALTER TABLE plan_indicator_reports
		ADD COLUMN IF NOT EXISTS reviewed_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL;
	`)
	if err != nil {
		return err
	}

	_, err = h.db.Exec(`
		ALTER TABLE plan_indicator_reports
		ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ NULL;
	`)
	if err != nil {
		return err
	}

	_, err = h.db.Exec(`
		UPDATE plan_indicator_reports
		SET status = 'pending'
		WHERE status IS NULL OR TRIM(status) = '';
	`)
	return err
}

func (h *Handler) ensurePlanIndicatorReportFilesTable() error {
	_, err := h.db.Exec(`
		CREATE TABLE IF NOT EXISTS plan_indicator_report_files (
			id BIGSERIAL PRIMARY KEY,
			report_id BIGINT NOT NULL REFERENCES plan_indicator_reports(id) ON DELETE CASCADE,
			file_name VARCHAR(255) NOT NULL,
			storage_path TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	if err != nil {
		return err
	}

	_, err = h.db.Exec(`
		ALTER TABLE plan_indicator_report_files
		ADD COLUMN IF NOT EXISTS storage_path TEXT NOT NULL DEFAULT '';
	`)
	if err != nil {
		return err
	}

	_, err = h.db.Exec(`
		INSERT INTO plan_indicator_report_files (report_id, file_name, storage_path, created_at)
		SELECT pir.id,
		       CASE
		           WHEN COALESCE(NULLIF(TRIM(pir.file_name), ''), '') <> '' THEN pir.file_name
		           ELSE CONCAT('legacy_file_', pir.id)
		       END,
		       pir.file_path,
		       NOW()
		FROM plan_indicator_reports pir
		WHERE COALESCE(TRIM(pir.file_path), '') <> ''
		  AND NOT EXISTS (
		      SELECT 1
		      FROM plan_indicator_report_files rf
		      WHERE rf.report_id = pir.id
		  );
	`)
	return err
}

func (h *Handler) saveIndicatorReport(
	c *gin.Context,
	indicatorID int,
	year int,
	reportText string,
	uploadedFiles []*multipart.FileHeader,
	submittedBy int64,
) (planIndicatorReportRow, error) {
	tx, err := h.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		return planIndicatorReportRow{}, err
	}
	defer tx.Rollback()

	reportID, oldFilePaths, err := h.upsertReportRow(tx, indicatorID, year, submittedBy, reportText)
	if err != nil {
		return planIndicatorReportRow{}, err
	}

	reportDir, err := resolvePlanReportStoragePath(indicatorID, year, submittedBy, reportID)
	if err != nil {
		return planIndicatorReportRow{}, err
	}
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		return planIndicatorReportRow{}, err
	}

	newFilePaths := make([]string, 0, len(uploadedFiles))
	for idx, fileHeader := range uploadedFiles {
		originalName := filepath.Base(fileHeader.Filename)
		safeName := sanitizeFileName(originalName)
		storedName := fmt.Sprintf("%d_%d_%s", time.Now().UnixNano(), idx+1, safeName)
		storagePath := filepath.Join(reportDir, storedName)

		if err := c.SaveUploadedFile(fileHeader, storagePath); err != nil {
			removeFiles(newFilePaths)
			return planIndicatorReportRow{}, err
		}
		newFilePaths = append(newFilePaths, storagePath)

		if _, err := tx.Exec(`
			INSERT INTO plan_indicator_report_files (report_id, file_name, storage_path, created_at)
			VALUES ($1, $2, $3, NOW())
		`, reportID, originalName, storagePath); err != nil {
			removeFiles(newFilePaths)
			return planIndicatorReportRow{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		removeFiles(newFilePaths)
		return planIndicatorReportRow{}, err
	}

	removeFiles(oldFilePaths)

	item, err := h.fetchPlanReportByID(reportID)
	if err != nil {
		return planIndicatorReportRow{}, err
	}
	return item, nil
}

func (h *Handler) upsertReportRow(
	tx *sql.Tx,
	indicatorID int,
	year int,
	submittedBy int64,
	reportText string,
) (int, []string, error) {
	var reportID int
	var currentStatus string
	err := tx.QueryRow(`
		SELECT id, status
		FROM plan_indicator_reports
		WHERE planning_period_indicator_id = $1
		  AND year = $2
		  AND submitted_by = $3
		LIMIT 1
	`, indicatorID, year, submittedBy).Scan(&reportID, &currentStatus)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, nil, err
	}

	if errors.Is(err, sql.ErrNoRows) {
		err = tx.QueryRow(`
			INSERT INTO plan_indicator_reports (
				planning_period_indicator_id,
				year,
				report_text,
				file_path,
				file_name,
				status,
				review_note,
				approval_formula,
				reviewed_by,
				reviewed_at,
				submitted_by,
				submitted_at,
				created_at,
				updated_at
			)
			VALUES ($1, $2, $3, '', '', $4, '', '', NULL, NULL, $5, NOW(), NOW(), NOW())
			RETURNING id
		`, indicatorID, year, reportText, reportStatusPending, submittedBy).Scan(&reportID)
		if err != nil {
			return 0, nil, err
		}
		return reportID, nil, nil
	}

	if strings.TrimSpace(currentStatus) != reportStatusRejected {
		return 0, nil, errReportAlreadySubmitted
	}

	oldFilePaths, err := collectReportFilePaths(tx, reportID)
	if err != nil {
		return 0, nil, err
	}

	if _, err := tx.Exec(`
		DELETE FROM plan_indicator_report_files
		WHERE report_id = $1
	`, reportID); err != nil {
		return 0, nil, err
	}

	if _, err := tx.Exec(`
		UPDATE plan_indicator_reports
		SET report_text = $1,
		    file_path = '',
		    file_name = '',
		    status = $2,
		    review_note = '',
		    approval_formula = '',
		    reviewed_by = NULL,
		    reviewed_at = NULL,
		    submitted_at = NOW(),
		    updated_at = NOW()
		WHERE id = $3
	`, reportText, reportStatusPending, reportID); err != nil {
		return 0, nil, err
	}

	return reportID, oldFilePaths, nil
}

func collectReportFilePaths(tx *sql.Tx, reportID int) ([]string, error) {
	rows, err := tx.Query(`
		SELECT storage_path
		FROM plan_indicator_report_files
		WHERE report_id = $1
	`, reportID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	paths := make([]string, 0)
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		path = strings.TrimSpace(path)
		if path != "" {
			paths = append(paths, path)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return paths, nil
}

func (h *Handler) fetchPlanReportByID(reportID int) (planIndicatorReportRow, error) {
	items, err := h.queryPlanReports("pir.id = $1", reportID)
	if err != nil {
		return planIndicatorReportRow{}, err
	}
	if len(items) == 0 {
		return planIndicatorReportRow{}, sql.ErrNoRows
	}
	return items[0], nil
}

func (h *Handler) queryPlanReports(whereClause string, args ...any) ([]planIndicatorReportRow, error) {
	query := `
		SELECT pir.id,
		       pir.planning_period_indicator_id,
		       pir.year,
		       COALESCE(NULLIF(pid.development_indicator, ''), ppi.target_indicator),
		       COALESCE(ppi.year_values ->> pir.year::text, ''),
		       COALESCE(ppi.unit, ''),
		       COALESCE(pid.execution_deadline, ''),
		       COALESCE(pid.responsible, ''),
		       COALESCE(pir.report_text, ''),
		       COALESCE(pir.status, 'pending'),
		       COALESCE(pir.review_note, ''),
		       COALESCE(pir.approval_formula, ''),
		       pir.submitted_by,
		       COALESCE(NULLIF(submitter.full_name, ''), submitter.username),
		       pir.submitted_at,
		       COALESCE(NULLIF(reviewer.full_name, ''), reviewer.username),
		       pir.reviewed_at
		FROM plan_indicator_reports pir
		JOIN planning_period_indicators ppi
		  ON ppi.id = pir.planning_period_indicator_id
		LEFT JOIN plan_indicator_details pid
		       ON pid.planning_period_indicator_id = pir.planning_period_indicator_id
		      AND pid.year = pir.year
		LEFT JOIN users submitter
		       ON submitter.id = pir.submitted_by
		LEFT JOIN users reviewer
		       ON reviewer.id = pir.reviewed_by
		WHERE ` + whereClause + `
		ORDER BY pir.id ASC
	`

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]planIndicatorReportRow, 0)
	for rows.Next() {
		var item planIndicatorReportRow
		var reviewedByName sql.NullString
		var reviewedAt sql.NullTime
		if err := rows.Scan(
			&item.ID,
			&item.IndicatorID,
			&item.Year,
			&item.DevelopmentIndicator,
			&item.PlannedValue,
			&item.Unit,
			&item.ExecutionDeadline,
			&item.Responsible,
			&item.ReportText,
			&item.Status,
			&item.ReviewNote,
			&item.ApprovalFormula,
			&item.SubmittedBy,
			&item.SubmittedByName,
			&item.SubmittedAt,
			&reviewedByName,
			&reviewedAt,
		); err != nil {
			return nil, err
		}

		if reviewedByName.Valid {
			item.ReviewedByName = reviewedByName.String
		}
		if reviewedAt.Valid {
			item.ReviewedAt = reviewedAt.Time
		}
		item.Files = []planIndicatorReportFile{}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := h.attachPlanReportFiles(items); err != nil {
		return nil, err
	}

	return items, nil
}

func (h *Handler) attachPlanReportFiles(items []planIndicatorReportRow) error {
	if len(items) == 0 {
		return nil
	}

	indices := make(map[int]int, len(items))
	ids := make([]int, 0, len(items))
	for idx := range items {
		reportID := items[idx].ID
		indices[reportID] = idx
		ids = append(ids, reportID)
		items[idx].Files = []planIndicatorReportFile{}
	}

	placeholders := make([]string, 0, len(ids))
	args := make([]any, 0, len(ids))
	for i, id := range ids {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
		args = append(args, id)
	}

	rows, err := h.db.Query(`
		SELECT id, report_id, file_name
		FROM plan_indicator_report_files
		WHERE report_id IN (`+strings.Join(placeholders, ",")+`)
		ORDER BY id ASC
	`, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var file planIndicatorReportFile
		var reportID int
		if err := rows.Scan(&file.ID, &reportID, &file.FileName); err != nil {
			return err
		}

		file.DownloadURL = fmt.Sprintf("/plans/reports/files/%d/download", file.ID)
		if index, ok := indices[reportID]; ok {
			items[index].Files = append(items[index].Files, file)
		}
	}

	return rows.Err()
}

func parseReportStatuses(raw string) ([]string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	allowed := map[string]struct{}{
		reportStatusPending:  {},
		reportStatusApproved: {},
		reportStatusRejected: {},
	}

	parts := strings.Split(trimmed, ",")
	statuses := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		status := strings.ToLower(strings.TrimSpace(part))
		if status == "" {
			continue
		}
		if _, ok := allowed[status]; !ok {
			return nil, fmt.Errorf("invalid status filter: %s", status)
		}
		if _, ok := seen[status]; ok {
			continue
		}
		seen[status] = struct{}{}
		statuses = append(statuses, status)
	}

	if len(statuses) == 0 {
		return nil, fmt.Errorf("status filter is empty")
	}

	return statuses, nil
}

func extractReportFiles(c *gin.Context) ([]*multipart.FileHeader, error) {
	form, err := c.MultipartForm()
	if err != nil {
		if errors.Is(err, http.ErrNotMultipart) {
			return nil, nil
		}
		return nil, err
	}

	filesList := make([]*multipart.FileHeader, 0)
	if form != nil && form.File != nil {
		filesList = append(filesList, form.File["files"]...)
	}

	if len(filesList) == 0 {
		single, singleErr := c.FormFile("file")
		if singleErr == nil && single != nil {
			filesList = append(filesList, single)
		}
	}

	return filesList, nil
}

func resolvePlanReportStoragePath(indicatorID int, year int, submittedBy int64, reportID int) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return filepath.Join(
		wd,
		"storage",
		"plan-reports",
		strconv.Itoa(indicatorID),
		strconv.Itoa(year),
		strconv.FormatInt(submittedBy, 10),
		strconv.Itoa(reportID),
	), nil
}

func sanitizeFileName(name string) string {
	base := strings.TrimSpace(filepath.Base(name))
	if base == "" {
		return "file"
	}

	base = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '.', r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, base)

	if base == "" {
		return "file"
	}
	return base
}

func removeFiles(paths []string) {
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		_ = os.Remove(path)
	}
}

func resolveExistingReportFilePath(storagePath string) (string, error) {
	trimmed := strings.TrimSpace(storagePath)
	if trimmed == "" {
		return "", os.ErrNotExist
	}

	candidates := make([]string, 0, 4)
	candidates = append(candidates, trimmed)

	wd, wdErr := os.Getwd()
	if wdErr == nil {
		if filepath.IsAbs(trimmed) {
			if strings.HasPrefix(trimmed, "/uploads/") {
				candidates = append(candidates, filepath.Join(wd, strings.TrimPrefix(trimmed, "/")))
			}
		} else {
			candidates = append(candidates, filepath.Join(wd, trimmed))
		}
	}

	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}

		stat, err := os.Stat(candidate)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", err
		}
		if stat.IsDir() {
			continue
		}
		return candidate, nil
	}

	return "", os.ErrNotExist
}

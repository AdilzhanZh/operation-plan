package plan

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
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
	reportStatusPending   = "pending"
	reportStatusCompleted = "completed"
	reportStatusRejected  = "rejected"
	reportStatusOverdue   = "overdue"
)

var errReportAlreadySubmitted = errors.New("report already submitted")

type listMeta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

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
	Meta  listMeta                 `json:"meta"`
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
	if err := h.ensureReportStatusHistoryTable(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to prepare report history storage"})
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

	var planItemID int64
	err = h.db.QueryRow(`
		SELECT pi.id
		FROM plan_items pi
		JOIN plan_item_responsibles pir
		  ON pir.plan_item_id = pi.id
		WHERE pi.indicator_id = $1
		  AND pi.year = $2
		  AND pir.user_id = $3
		LIMIT 1
	`, indicatorID, year, user.ID).Scan(&planItemID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(403, errorResponse{Error: "indicator is not assigned to this prorector"})
			return
		}
		c.JSON(500, errorResponse{Error: "failed to validate responsibility"})
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

	item, err := h.saveIndicatorReport(c, planItemID, indicatorID, year, reportText, uploadedFiles, user.ID)
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
// @Param status query string false "Optional statuses: pending,completed,rejected,overdue. Comma-separated."
// @Param q query string false "Search text by indicator/report"
// @Param submitted_by query int false "Filter by prorector ID (admin only)"
// @Param page query int false "Page number (default 1)"
// @Param limit query int false "Items per page (default 20, max 100)"
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
	page, limit, err := parsePagination(c.Query("page"), c.Query("limit"))
	if err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}
	statuses, err := parseReportStatuses(c.Query("status"))
	if err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}
	searchQuery := strings.TrimSpace(c.Query("q"))

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

	args := make([]any, 0, 10+len(statuses))
	where := make([]string, 0, 10)
	args = append(args, year)
	where = append(where, "pi.year = $1")

	if user.Role == "prorector" {
		args = append(args, user.ID)
		where = append(where, fmt.Sprintf("rs.submitted_by = $%d", len(args)))
	} else {
		submittedByRaw := strings.TrimSpace(c.Query("submitted_by"))
		if submittedByRaw != "" {
			submittedBy, convErr := strconv.ParseInt(submittedByRaw, 10, 64)
			if convErr != nil || submittedBy <= 0 {
				c.JSON(400, errorResponse{Error: "submitted_by must be positive integer"})
				return
			}
			args = append(args, submittedBy)
			where = append(where, fmt.Sprintf("rs.submitted_by = $%d", len(args)))
		}
	}

	if len(statuses) > 0 {
		statusCondition, statusArgs := buildReportStatusCondition(statuses, len(args))
		args = append(args, statusArgs...)
		where = append(where, statusCondition)
	}

	if searchQuery != "" {
		args = append(args, "%"+searchQuery+"%")
		ph := fmt.Sprintf("$%d", len(args))
		where = append(where, fmt.Sprintf(`(
			COALESCE(NULLIF(pi.development_indicator, ''), ppi.target_indicator) ILIKE %s
			OR COALESCE(rs.report_text, '') ILIKE %s
			OR EXISTS (
				SELECT 1
				FROM plan_item_responsibles pir_s
				JOIN users u_s ON u_s.id = pir_s.user_id
				WHERE pir_s.plan_item_id = pi.id
				  AND COALESCE(NULLIF(TRIM(u_s.full_name), ''), u_s.username) ILIKE %s
			)
		)`, ph, ph, ph))
	}

	whereClause := strings.Join(where, " AND ")
	total, err := h.countPlanReports(whereClause, args...)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to count reports"})
		return
	}

	items, err := h.queryPlanReportsPaged(whereClause, page, limit, args...)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to load reports"})
		return
	}

	totalPages := total / limit
	if total%limit != 0 {
		totalPages++
	}

	c.JSON(200, planIndicatorReportListResponse{
		Year:  year,
		Items: items,
		Meta: listMeta{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
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
// @Failure 409 {object} errorResponse
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
	if err := h.ensureReportStatusHistoryTable(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to prepare report history storage"})
		return
	}

	var currentStatus string
	err = h.db.QueryRow(`
		SELECT COALESCE(status, 'pending')
		FROM report_submissions
		WHERE id = $1
		LIMIT 1
	`, reportID).Scan(&currentStatus)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(404, errorResponse{Error: "report not found"})
			return
		}
		c.JSON(500, errorResponse{Error: "failed to load report state"})
		return
	}

	action := strings.ToLower(strings.TrimSpace(req.Action))
	targetStatus := ""
	reviewNote := ""
	approvalFormula := ""
	historyNote := ""

	switch action {
	case "approve":
		approvalFormula = strings.TrimSpace(req.ApprovalFormula)
		if approvalFormula == "" {
			c.JSON(400, errorResponse{Error: "approval_formula is required"})
			return
		}
		targetStatus = reportStatusCompleted
		historyNote = approvalFormula
	case "reject":
		reviewNote = strings.TrimSpace(req.ReviewNote)
		if reviewNote == "" {
			c.JSON(400, errorResponse{Error: "review_note is required"})
			return
		}
		targetStatus = reportStatusRejected
		historyNote = reviewNote
	default:
		c.JSON(400, errorResponse{Error: "action must be approve or reject"})
		return
	}

	if !canReportStatusTransition(currentStatus, targetStatus) {
		c.JSON(409, errorResponse{Error: fmt.Sprintf("invalid state transition: %s -> %s", normalizeReportStatus(currentStatus), targetStatus)})
		return
	}

	tx, err := h.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to start transaction"})
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		UPDATE report_submissions
		SET status = $1,
		    review_note = $2,
		    approval_formula = $3,
		    reviewed_by = $4,
		    reviewed_at = NOW(),
		    updated_at = NOW()
		WHERE id = $5
	`, targetStatus, reviewNote, approvalFormula, user.ID, reportID)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to review report"})
		return
	}

	if err := insertReportStatusHistoryTx(tx, int64(reportID), normalizeReportStatus(currentStatus), targetStatus, user.ID, historyNote); err != nil {
		c.JSON(500, errorResponse{Error: "failed to record report history"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to commit review"})
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
	var storageKey string
	var submittedBy int64
	err = h.db.QueryRow(`
		SELECT rf.file_name,
		       rf.storage_key,
		       rs.submitted_by
		FROM report_files rf
		JOIN report_submissions rs
		  ON rs.id = rf.submission_id
		WHERE rf.id = $1
		LIMIT 1
	`, fileID).Scan(&fileName, &storageKey, &submittedBy)
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

	finalPath, err := resolveExistingReportFilePath(storageKey)
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
	h.ensureReportSubmissionsOnce.Do(func() {
		statements := []string{
			`CREATE TABLE IF NOT EXISTS report_submissions (
				id BIGSERIAL PRIMARY KEY,
				plan_item_id BIGINT NOT NULL REFERENCES plan_items(id) ON DELETE CASCADE,
				submitted_by BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				report_text TEXT NOT NULL DEFAULT '',
				status VARCHAR(32) NOT NULL DEFAULT 'pending',
				review_note TEXT NOT NULL DEFAULT '',
				approval_formula TEXT NOT NULL DEFAULT '',
				reviewed_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
				submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				reviewed_at TIMESTAMPTZ NULL,
				created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				UNIQUE (plan_item_id, submitted_by),
				CHECK (status IN ('pending', 'completed', 'rejected'))
			);`,
			`CREATE INDEX IF NOT EXISTS report_submissions_status_idx ON report_submissions (status);`,
			`CREATE INDEX IF NOT EXISTS report_submissions_plan_item_idx ON report_submissions (plan_item_id);`,
			`UPDATE report_submissions
			SET status = LOWER(TRIM(COALESCE(status, '')));`,
			`UPDATE report_submissions
			SET status = 'pending'
			WHERE status = '';`,
			`UPDATE report_submissions
			SET status = 'completed'
			WHERE status = 'approved';`,
			`UPDATE report_submissions
			SET status = 'pending'
			WHERE status NOT IN ('pending', 'completed', 'rejected');`,
			`DO $$
			BEGIN
			  IF EXISTS (
			    SELECT 1
			    FROM pg_constraint
			    WHERE conname = 'report_submissions_status_check'
			  ) THEN
			    ALTER TABLE report_submissions
			      DROP CONSTRAINT report_submissions_status_check;
			  END IF;
			END $$;`,
			`DO $$
			BEGIN
			  IF NOT EXISTS (
			    SELECT 1
			    FROM pg_constraint
			    WHERE conname = 'report_submissions_status_check'
			  ) THEN
			    ALTER TABLE report_submissions
			      ADD CONSTRAINT report_submissions_status_check
			      CHECK (status IN ('pending', 'completed', 'rejected'));
			  END IF;
			END $$;`,
			`INSERT INTO report_submissions (
				plan_item_id,
				submitted_by,
				report_text,
				status,
				review_note,
				approval_formula,
				reviewed_by,
				submitted_at,
				reviewed_at,
				created_at,
				updated_at
			)
			SELECT pi.id,
			       pir.submitted_by,
			       COALESCE(pir.report_text, ''),
			       CASE
			           WHEN LOWER(TRIM(COALESCE(pir.status, ''))) IN ('completed', 'approved')
			           THEN 'completed'
			           WHEN LOWER(TRIM(COALESCE(pir.status, ''))) = 'rejected'
			           THEN 'rejected'
			           ELSE 'pending'
			       END,
			       COALESCE(pir.review_note, ''),
			       COALESCE(pir.approval_formula, ''),
			       pir.reviewed_by,
			       COALESCE(pir.submitted_at, NOW()),
			       pir.reviewed_at,
			       COALESCE(pir.created_at, NOW()),
			       COALESCE(pir.updated_at, NOW())
			FROM plan_indicator_reports pir
			JOIN plan_items pi
			  ON pi.indicator_id = pir.planning_period_indicator_id
			 AND pi.year = pir.year
			ON CONFLICT (plan_item_id, submitted_by)
			DO UPDATE SET
				report_text = EXCLUDED.report_text,
				status = EXCLUDED.status,
				review_note = EXCLUDED.review_note,
				approval_formula = EXCLUDED.approval_formula,
				reviewed_by = EXCLUDED.reviewed_by,
				submitted_at = EXCLUDED.submitted_at,
				reviewed_at = EXCLUDED.reviewed_at,
				updated_at = NOW();`,
		}

		for _, stmt := range statements {
			if _, err := h.db.Exec(stmt); err != nil {
				h.ensureReportSubmissionsErr = err
				return
			}
		}
	})

	return h.ensureReportSubmissionsErr
}

func (h *Handler) ensurePlanIndicatorReportFilesTable() error {
	h.ensureReportFilesOnce.Do(func() {
		statements := []string{
			`CREATE TABLE IF NOT EXISTS report_files (
				id BIGSERIAL PRIMARY KEY,
				submission_id BIGINT NOT NULL REFERENCES report_submissions(id) ON DELETE CASCADE,
				file_name VARCHAR(255) NOT NULL,
				storage_key TEXT NOT NULL,
				mime_type VARCHAR(255) NOT NULL DEFAULT '',
				file_size BIGINT NOT NULL DEFAULT 0,
				sha256 VARCHAR(64) NOT NULL DEFAULT '',
				created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
			);`,
			`CREATE INDEX IF NOT EXISTS report_files_submission_idx ON report_files (submission_id);`,
			`CREATE UNIQUE INDEX IF NOT EXISTS report_files_submission_storage_uindex ON report_files (submission_id, storage_key);`,
			`INSERT INTO report_files (submission_id, file_name, storage_key, mime_type, file_size, sha256, created_at)
			SELECT rs.id,
			       COALESCE(NULLIF(TRIM(prf.file_name), ''), CONCAT('file_', prf.id)),
			       COALESCE(prf.storage_path, ''),
			       '',
			       0,
			       '',
			       COALESCE(prf.created_at, NOW())
			FROM plan_indicator_report_files prf
			JOIN plan_indicator_reports pir
			  ON pir.id = prf.report_id
			JOIN plan_items pi
			  ON pi.indicator_id = pir.planning_period_indicator_id
			 AND pi.year = pir.year
			JOIN report_submissions rs
			  ON rs.plan_item_id = pi.id
			 AND rs.submitted_by = pir.submitted_by
			WHERE COALESCE(prf.storage_path, '') <> ''
			ON CONFLICT (submission_id, storage_key) DO NOTHING;`,
			`INSERT INTO report_files (submission_id, file_name, storage_key, mime_type, file_size, sha256, created_at)
			SELECT rs.id,
			       CASE
			           WHEN COALESCE(NULLIF(TRIM(pir.file_name), ''), '') <> '' THEN pir.file_name
			           ELSE CONCAT('legacy_file_', pir.id)
			       END,
			       pir.file_path,
			       '',
			       0,
			       '',
			       NOW()
			FROM plan_indicator_reports pir
			JOIN plan_items pi
			  ON pi.indicator_id = pir.planning_period_indicator_id
			 AND pi.year = pir.year
			JOIN report_submissions rs
			  ON rs.plan_item_id = pi.id
			 AND rs.submitted_by = pir.submitted_by
			WHERE COALESCE(TRIM(pir.file_path), '') <> ''
			ON CONFLICT (submission_id, storage_key) DO NOTHING;`,
		}

		for _, stmt := range statements {
			if _, err := h.db.Exec(stmt); err != nil {
				h.ensureReportFilesErr = err
				return
			}
		}
	})

	return h.ensureReportFilesErr
}

func (h *Handler) ensureReportStatusHistoryTable() error {
	h.ensureReportHistoryOnce.Do(func() {
		statements := []string{
			`CREATE TABLE IF NOT EXISTS report_status_history (
				id BIGSERIAL PRIMARY KEY,
				submission_id BIGINT NOT NULL REFERENCES report_submissions(id) ON DELETE CASCADE,
				from_status VARCHAR(32) NOT NULL DEFAULT '',
				to_status VARCHAR(32) NOT NULL,
				actor_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
				note TEXT NOT NULL DEFAULT '',
				created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
			);`,
			`CREATE INDEX IF NOT EXISTS report_status_history_submission_idx ON report_status_history (submission_id, created_at DESC);`,
			`INSERT INTO report_status_history (submission_id, from_status, to_status, actor_id, note, created_at)
			SELECT rs.id,
			       '',
			       rs.status,
			       rs.submitted_by,
			       'initial migration',
			       COALESCE(rs.submitted_at, NOW())
			FROM report_submissions rs
			WHERE NOT EXISTS (
				SELECT 1
				FROM report_status_history rsh
				WHERE rsh.submission_id = rs.id
			);`,
		}

		for _, stmt := range statements {
			if _, err := h.db.Exec(stmt); err != nil {
				h.ensureReportHistoryErr = err
				return
			}
		}
	})

	return h.ensureReportHistoryErr
}

func (h *Handler) saveIndicatorReport(
	c *gin.Context,
	planItemID int64,
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

	submissionID, oldFilePaths, err := h.upsertReportRow(tx, planItemID, submittedBy, reportText)
	if err != nil {
		return planIndicatorReportRow{}, err
	}

	reportDir, err := resolvePlanReportStoragePath(indicatorID, year, submittedBy, int(submissionID))
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

		sha256Hex, hashErr := computeFileSHA256(storagePath)
		if hashErr != nil {
			removeFiles(newFilePaths)
			return planIndicatorReportRow{}, hashErr
		}

		mimeType := strings.TrimSpace(fileHeader.Header.Get("Content-Type"))
		if _, err := tx.Exec(`
			INSERT INTO report_files (
				submission_id,
				file_name,
				storage_key,
				mime_type,
				file_size,
				sha256,
				created_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, NOW())
			ON CONFLICT (submission_id, storage_key) DO NOTHING
		`, submissionID, originalName, storagePath, mimeType, fileHeader.Size, sha256Hex); err != nil {
			removeFiles(newFilePaths)
			return planIndicatorReportRow{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		removeFiles(newFilePaths)
		return planIndicatorReportRow{}, err
	}

	removeFiles(oldFilePaths)

	item, err := h.fetchPlanReportByID(int(submissionID))
	if err != nil {
		return planIndicatorReportRow{}, err
	}
	return item, nil
}

func (h *Handler) upsertReportRow(
	tx *sql.Tx,
	planItemID int64,
	submittedBy int64,
	reportText string,
) (int64, []string, error) {
	var submissionID int64
	var currentStatus string
	err := tx.QueryRow(`
		SELECT id, status
		FROM report_submissions
		WHERE plan_item_id = $1
		  AND submitted_by = $2
		LIMIT 1
	`, planItemID, submittedBy).Scan(&submissionID, &currentStatus)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, nil, err
	}

	if errors.Is(err, sql.ErrNoRows) {
		err = tx.QueryRow(`
			INSERT INTO report_submissions (
				plan_item_id,
				submitted_by,
				report_text,
				status,
				review_note,
				approval_formula,
				reviewed_by,
				submitted_at,
				created_at,
				updated_at
			)
			VALUES ($1, $2, $3, $4, '', '', NULL, NOW(), NOW(), NOW())
			RETURNING id
		`, planItemID, submittedBy, reportText, reportStatusPending).Scan(&submissionID)
		if err != nil {
			return 0, nil, err
		}

		if err := insertReportStatusHistoryTx(tx, submissionID, "", reportStatusPending, submittedBy, "submitted"); err != nil {
			return 0, nil, err
		}
		return submissionID, nil, nil
	}

	if !canReportStatusTransition(currentStatus, reportStatusPending) {
		return 0, nil, errReportAlreadySubmitted
	}

	oldFilePaths, err := collectSubmissionFilePaths(tx, submissionID)
	if err != nil {
		return 0, nil, err
	}

	if _, err := tx.Exec(`
		DELETE FROM report_files
		WHERE submission_id = $1
	`, submissionID); err != nil {
		return 0, nil, err
	}

	if _, err := tx.Exec(`
		UPDATE report_submissions
		SET report_text = $1,
		    status = $2,
		    review_note = '',
		    approval_formula = '',
		    reviewed_by = NULL,
		    reviewed_at = NULL,
		    submitted_at = NOW(),
		    updated_at = NOW()
		WHERE id = $3
	`, reportText, reportStatusPending, submissionID); err != nil {
		return 0, nil, err
	}

	if err := insertReportStatusHistoryTx(tx, submissionID, normalizeReportStatus(currentStatus), reportStatusPending, submittedBy, "resubmitted"); err != nil {
		return 0, nil, err
	}

	return submissionID, oldFilePaths, nil
}

func collectSubmissionFilePaths(tx *sql.Tx, submissionID int64) ([]string, error) {
	rows, err := tx.Query(`
		SELECT storage_key
		FROM report_files
		WHERE submission_id = $1
	`, submissionID)
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
	items, err := h.queryPlanReportsPaged("rs.id = $1", 1, 1, reportID)
	if err != nil {
		return planIndicatorReportRow{}, err
	}
	if len(items) == 0 {
		return planIndicatorReportRow{}, sql.ErrNoRows
	}
	return items[0], nil
}

func (h *Handler) queryPlanReportsPaged(whereClause string, page int, limit int, args ...any) ([]planIndicatorReportRow, error) {
	baseQuery := buildPlanReportsBaseQuery(whereClause)
	queryArgs := append([]any{}, args...)
	offset := (page - 1) * limit
	queryArgs = append(queryArgs, limit, offset)

	query := `
		SELECT rs.id,
		       pi.indicator_id,
		       pi.year,
		       COALESCE(NULLIF(pi.development_indicator, ''), ppi.target_indicator),
		       COALESCE(iyt.planned_value, ''),
		       COALESCE(ppi.unit, ''),
		       CASE
		           WHEN pi.execution_start_date IS NOT NULL AND pi.execution_end_date IS NOT NULL
		           THEN TO_CHAR(pi.execution_start_date, 'DD.MM.YYYY') || ' - ' || TO_CHAR(pi.execution_end_date, 'DD.MM.YYYY')
		           ELSE ''
		       END,
		       COALESCE((
		           SELECT STRING_AGG(
		              COALESCE(NULLIF(TRIM(u.full_name), ''), u.username),
		              ', '
		              ORDER BY COALESCE(NULLIF(TRIM(u.full_name), ''), u.username)
		           )
		           FROM plan_item_responsibles pir
		           JOIN users u ON u.id = pir.user_id
		           WHERE pir.plan_item_id = pi.id
		       ), ''),
		       COALESCE(rs.report_text, ''),
		       COALESCE(rs.status, 'pending'),
		       COALESCE(rs.review_note, ''),
		       COALESCE(rs.approval_formula, ''),
		       rs.submitted_by,
		       COALESCE(NULLIF(submitter.full_name, ''), submitter.username),
		       rs.submitted_at,
		       COALESCE(NULLIF(reviewer.full_name, ''), reviewer.username),
		       rs.reviewed_at
	` + baseQuery + fmt.Sprintf(`
		ORDER BY rs.id ASC
		LIMIT $%d
		OFFSET $%d
	`, len(queryArgs)-1, len(queryArgs))

	rows, err := h.db.Query(query, queryArgs...)
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

func (h *Handler) countPlanReports(whereClause string, args ...any) (int, error) {
	var total int
	err := h.db.QueryRow(`
		SELECT COUNT(*)
	`+buildPlanReportsBaseQuery(whereClause), args...).Scan(&total)
	if err != nil {
		return 0, err
	}
	return total, nil
}

func (h *Handler) attachPlanReportFiles(items []planIndicatorReportRow) error {
	if len(items) == 0 {
		return nil
	}

	indices := make(map[int]int, len(items))
	ids := make([]int, 0, len(items))
	for idx := range items {
		indices[items[idx].ID] = idx
		ids = append(ids, items[idx].ID)
		items[idx].Files = []planIndicatorReportFile{}
	}

	placeholders := make([]string, 0, len(ids))
	args := make([]any, 0, len(ids))
	for i, id := range ids {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
		args = append(args, id)
	}

	rows, err := h.db.Query(`
		SELECT id, submission_id, file_name
		FROM report_files
		WHERE submission_id IN (`+strings.Join(placeholders, ",")+`)
		ORDER BY id ASC
	`, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var file planIndicatorReportFile
		var submissionID int
		if err := rows.Scan(&file.ID, &submissionID, &file.FileName); err != nil {
			return err
		}
		file.DownloadURL = fmt.Sprintf("/plans/reports/files/%d/download", file.ID)
		if idx, ok := indices[submissionID]; ok {
			items[idx].Files = append(items[idx].Files, file)
		}
	}

	return rows.Err()
}

func buildPlanReportsBaseQuery(whereClause string) string {
	return `
		FROM report_submissions rs
		JOIN plan_items pi
		  ON pi.id = rs.plan_item_id
		JOIN planning_period_indicators ppi
		  ON ppi.id = pi.indicator_id
		LEFT JOIN indicator_year_targets iyt
		       ON iyt.indicator_id = pi.indicator_id
		      AND iyt.year = pi.year
		LEFT JOIN users submitter
		       ON submitter.id = rs.submitted_by
		LEFT JOIN users reviewer
		       ON reviewer.id = rs.reviewed_by
		WHERE ` + whereClause + `
	`
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

func normalizeReportStatus(status string) string {
	return strings.ToLower(strings.TrimSpace(status))
}

func canReportStatusTransition(fromStatus, toStatus string) bool {
	from := normalizeReportStatus(fromStatus)
	to := normalizeReportStatus(toStatus)

	switch from {
	case "":
		return to == reportStatusPending
	case reportStatusPending:
		return to == reportStatusCompleted || to == reportStatusRejected
	case reportStatusRejected:
		return to == reportStatusPending
	case reportStatusCompleted:
		return false
	default:
		return false
	}
}

func buildReportStatusCondition(statuses []string, initialArgCount int) (string, []any) {
	normalStatuses := make([]string, 0, len(statuses))
	includeOverdue := false
	for _, status := range statuses {
		if status == reportStatusOverdue {
			includeOverdue = true
			continue
		}
		normalStatuses = append(normalStatuses, status)
	}

	clauses := make([]string, 0, 2)
	args := make([]any, 0, len(normalStatuses))

	if len(normalStatuses) > 0 {
		placeholders := make([]string, 0, len(normalStatuses))
		for _, status := range normalStatuses {
			args = append(args, status)
			placeholders = append(placeholders, fmt.Sprintf("$%d", initialArgCount+len(args)))
		}
		clauses = append(clauses, fmt.Sprintf(
			"COALESCE(rs.status, 'pending') IN (%s)",
			strings.Join(placeholders, ", "),
		))
	}

	if includeOverdue {
		clauses = append(clauses, `(
			COALESCE(rs.status, 'pending') <> 'completed'
			AND pi.execution_end_date IS NOT NULL
			AND pi.execution_end_date < CURRENT_DATE
		)`)
	}

	if len(clauses) == 0 {
		return "1=1", nil
	}

	return "(" + strings.Join(clauses, " OR ") + ")", args
}

func parseReportStatuses(raw string) ([]string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	allowed := map[string]struct{}{
		reportStatusPending:   {},
		reportStatusCompleted: {},
		reportStatusRejected:  {},
		reportStatusOverdue:   {},
	}

	parts := strings.Split(trimmed, ",")
	statuses := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		status := strings.ToLower(strings.TrimSpace(part))
		if status == "" {
			continue
		}
		if status == "approved" {
			status = reportStatusCompleted
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

func computeFileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func insertReportStatusHistoryTx(tx *sql.Tx, submissionID int64, fromStatus string, toStatus string, actorID int64, note string) error {
	_, err := tx.Exec(`
		INSERT INTO report_status_history (
			submission_id,
			from_status,
			to_status,
			actor_id,
			note,
			created_at
		)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`, submissionID, strings.TrimSpace(fromStatus), strings.TrimSpace(toStatus), actorID, strings.TrimSpace(note))
	return err
}

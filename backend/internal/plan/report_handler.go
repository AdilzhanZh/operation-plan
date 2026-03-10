package plan

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"OperationPlan/internal/files"
	"OperationPlan/internal/middleware"

	"github.com/gin-gonic/gin"
)

type planIndicatorReportRow struct {
	ID                   int       `json:"id"`
	IndicatorID          int       `json:"indicator_id"`
	Year                 int       `json:"year"`
	DevelopmentIndicator string    `json:"development_indicator"`
	PlannedValue         string    `json:"planned_value"`
	Unit                 string    `json:"unit"`
	ExecutionDeadline    string    `json:"execution_deadline"`
	Responsible          string    `json:"responsible"`
	ReportText           string    `json:"report_text"`
	FilePath             string    `json:"file_path"`
	FileName             string    `json:"file_name"`
	SubmittedBy          int64     `json:"submitted_by"`
	SubmittedByName      string    `json:"submitted_by_name"`
	SubmittedAt          time.Time `json:"submitted_at"`
}

type planIndicatorReportListResponse struct {
	Year  int                      `json:"year"`
	Items []planIndicatorReportRow `json:"items"`
}

// submitPlanIndicatorReport godoc
// @Summary Submit indicator report
// @Description Prorector submits report text and optional file for assigned indicator
// @Tags plans
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param indicator_id path int true "Planning period indicator ID"
// @Param year query int true "Year"
// @Param report_text formData string false "Report text"
// @Param file formData file false "Supporting file"
// @Success 201 {object} planIndicatorReportRow
// @Failure 400 {object} errorResponse
// @Failure 403 {object} errorResponse
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

	var filePath string
	var fileName string
	fileHeader, fileErr := c.FormFile("file")
	if fileErr == nil && fileHeader != nil {
		if err := files.ValidateUpload(fileHeader); err != nil {
			c.JSON(400, errorResponse{Error: err.Error()})
			return
		}

		fileName = filepath.Base(fileHeader.Filename)
		filePath = fmt.Sprintf("/uploads/plan-indicators/%d/%d/%s", indicatorID, year, fileName)
	} else if fileErr != nil && !errors.Is(fileErr, http.ErrMissingFile) {
		c.JSON(400, errorResponse{Error: "invalid file"})
		return
	}

	if reportText == "" && filePath == "" {
		c.JSON(400, errorResponse{Error: "report_text or file is required"})
		return
	}

	var item planIndicatorReportRow
	err = h.db.QueryRow(`
		INSERT INTO plan_indicator_reports (
			planning_period_indicator_id,
			year,
			report_text,
			file_path,
			file_name,
			submitted_by,
			submitted_at,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW(), NOW())
		ON CONFLICT (planning_period_indicator_id, year, submitted_by)
		DO UPDATE SET
			report_text = EXCLUDED.report_text,
			file_path = CASE
				WHEN EXCLUDED.file_path <> '' THEN EXCLUDED.file_path
				ELSE plan_indicator_reports.file_path
			END,
			file_name = CASE
				WHEN EXCLUDED.file_name <> '' THEN EXCLUDED.file_name
				ELSE plan_indicator_reports.file_name
			END,
			submitted_at = NOW(),
			updated_at = NOW()
		RETURNING id,
		          planning_period_indicator_id,
		          year,
		          report_text,
		          file_path,
		          file_name,
		          submitted_by,
		          submitted_at
	`,
		indicatorID,
		year,
		reportText,
		filePath,
		fileName,
		user.ID,
	).Scan(
		&item.ID,
		&item.IndicatorID,
		&item.Year,
		&item.ReportText,
		&item.FilePath,
		&item.FileName,
		&item.SubmittedBy,
		&item.SubmittedAt,
	)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to save report"})
		return
	}

	err = h.db.QueryRow(`
		SELECT COALESCE(NULLIF(pid.development_indicator, ''), ppi.target_indicator),
		       COALESCE(ppi.year_values ->> $1, ''),
		       COALESCE(ppi.unit, ''),
		       COALESCE(pid.execution_deadline, ''),
		       COALESCE(pid.responsible, ''),
		       COALESCE(NULLIF(u.full_name, ''), u.username)
		FROM planning_period_indicators ppi
		LEFT JOIN plan_indicator_details pid
		       ON pid.planning_period_indicator_id = ppi.id
		      AND pid.year = $2
		LEFT JOIN users u
		       ON u.id = $4
		WHERE ppi.id = $3
		LIMIT 1
	`, yearKey, year, indicatorID, user.ID).Scan(
		&item.DevelopmentIndicator,
		&item.PlannedValue,
		&item.Unit,
		&item.ExecutionDeadline,
		&item.Responsible,
		&item.SubmittedByName,
	)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to load saved report"})
		return
	}

	c.JSON(201, item)
}

// listPlanReports godoc
// @Summary List submitted indicator reports
// @Description Returns prorector submissions for selected year
// @Tags plans
// @Produce json
// @Security BearerAuth
// @Param year query int true "Year"
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
	if user.Role != "admin" {
		c.JSON(403, errorResponse{Error: "forbidden"})
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

	rows, err := h.db.Query(`
		SELECT pir.id,
		       pir.planning_period_indicator_id,
		       pir.year,
		       COALESCE(NULLIF(pid.development_indicator, ''), ppi.target_indicator),
		       COALESCE(ppi.year_values ->> $1, ''),
		       COALESCE(ppi.unit, ''),
		       COALESCE(pid.execution_deadline, ''),
		       COALESCE(pid.responsible, ''),
		       pir.report_text,
		       pir.file_path,
		       pir.file_name,
		       pir.submitted_by,
		       COALESCE(NULLIF(u.full_name, ''), u.username),
		       pir.submitted_at
		FROM plan_indicator_reports pir
		JOIN planning_period_indicators ppi
		  ON ppi.id = pir.planning_period_indicator_id
		LEFT JOIN plan_indicator_details pid
		       ON pid.planning_period_indicator_id = pir.planning_period_indicator_id
		      AND pid.year = pir.year
		LEFT JOIN users u
		       ON u.id = pir.submitted_by
		WHERE pir.year = $2
		ORDER BY pir.id ASC
	`, yearKey, year)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to load reports"})
		return
	}
	defer rows.Close()

	items := make([]planIndicatorReportRow, 0)
	for rows.Next() {
		var item planIndicatorReportRow
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
			&item.FilePath,
			&item.FileName,
			&item.SubmittedBy,
			&item.SubmittedByName,
			&item.SubmittedAt,
		); err != nil {
			c.JSON(500, errorResponse{Error: "failed to parse reports"})
			return
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to iterate reports"})
		return
	}

	c.JSON(200, planIndicatorReportListResponse{
		Year:  year,
		Items: items,
	})
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
	return err
}

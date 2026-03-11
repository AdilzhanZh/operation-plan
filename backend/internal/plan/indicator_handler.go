package plan

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"OperationPlan/internal/middleware"

	"github.com/gin-gonic/gin"
)

type planYearsResponse struct {
	Years []int `json:"years"`
}

type planIndicatorListResponse struct {
	Year  int                `json:"year"`
	Items []planIndicatorRow `json:"items"`
	Meta  listMeta           `json:"meta"`
}

type planIndicatorRow struct {
	IndicatorID          int     `json:"indicator_id"`
	SourceIndicator      string  `json:"source_indicator"`
	Unit                 string  `json:"unit"`
	MeasurementUnit      string  `json:"measurement_unit"`
	PlannedValue         string  `json:"planned_value"`
	DevelopmentIndicator string  `json:"development_indicator"`
	Activities           string  `json:"activities"`
	ExecutionDeadline    string  `json:"execution_deadline"`
	ExecutionStartDate   string  `json:"execution_start_date"`
	ExecutionEndDate     string  `json:"execution_end_date"`
	ScheduleStatus       string  `json:"schedule_status"`
	ReportStatus         string  `json:"report_status"`
	LastSubmittedAt      string  `json:"last_submitted_at"`
	Responsible          string  `json:"responsible"`
	ResponsibleUserIDs   []int64 `json:"responsible_user_ids"`
}

type upsertPlanIndicatorRequest struct {
	DevelopmentIndicator string  `json:"development_indicator"`
	Activities           string  `json:"activities"`
	ExecutionDeadline    string  `json:"execution_deadline"`
	ExecutionStartDate   string  `json:"execution_start_date"`
	ExecutionEndDate     string  `json:"execution_end_date"`
	ResponsibleUserIDs   []int64 `json:"responsible_user_ids"`
}

// listPlanYears godoc
// @Summary List available plan years
// @Description Returns years extracted from normalized planning targets
// @Tags plans
// @Produce json
// @Security BearerAuth
// @Success 200 {object} planYearsResponse
// @Failure 500 {object} errorResponse
// @Router /plans/years [get]
func (h *Handler) listPlanYears(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(401, errorResponse{Error: "unauthorized"})
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

	var (
		rows *sql.Rows
		err  error
	)
	if user.Role == "prorector" {
		rows, err = h.db.Query(`
			SELECT DISTINCT pi.year
			FROM plan_items pi
			JOIN plan_item_responsibles pir
			  ON pir.plan_item_id = pi.id
			WHERE pir.user_id = $1
			ORDER BY pi.year ASC
		`, user.ID)
	} else {
		rows, err = h.db.Query(`
			SELECT DISTINCT year
			FROM indicator_year_targets
			ORDER BY year ASC
		`)
	}
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to load planning years"})
		return
	}
	defer rows.Close()

	years := make([]int, 0)
	for rows.Next() {
		var year int
		if err := rows.Scan(&year); err != nil {
			c.JSON(500, errorResponse{Error: "failed to parse planning years"})
			return
		}
		years = append(years, year)
	}
	if err := rows.Err(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to iterate planning years"})
		return
	}

	c.JSON(200, planYearsResponse{Years: years})
}

// listPlanIndicators godoc
// @Summary List plan indicators for selected year
// @Description Returns plan rows based on normalized planning targets for selected year
// @Tags plans
// @Produce json
// @Security BearerAuth
// @Param year query int true "Year"
// @Param q query string false "Search text"
// @Param page query int false "Page number (default 1)"
// @Param limit query int false "Items per page (default 20, max 100)"
// @Success 200 {object} planIndicatorListResponse
// @Failure 400 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /plans/indicators [get]
func (h *Handler) listPlanIndicators(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(401, errorResponse{Error: "unauthorized"})
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
	searchQuery := strings.TrimSpace(c.Query("q"))
	includeSubmitted := strings.EqualFold(strings.TrimSpace(c.Query("include_submitted")), "true")

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

	args := make([]any, 0, 6)
	where := make([]string, 0, 6)
	args = append(args, year)
	where = append(where, "iyt.year = $1")

	if searchQuery != "" {
		args = append(args, "%"+searchQuery+"%")
		ph := fmt.Sprintf("$%d", len(args))
		where = append(where, fmt.Sprintf(`(
			ppi.target_indicator ILIKE %s
			OR COALESCE(pi.development_indicator, '') ILIKE %s
			OR COALESCE(pi.activities, '') ILIKE %s
		)`, ph, ph, ph))
	}

	if user.Role == "prorector" {
		args = append(args, user.ID)
		ph := fmt.Sprintf("$%d", len(args))
		where = append(where, fmt.Sprintf(`
			EXISTS (
				SELECT 1
				FROM plan_item_responsibles pir_self
				WHERE pir_self.plan_item_id = pi.id
				  AND pir_self.user_id = %s
			)
		`, ph))
		if !includeSubmitted {
			where = append(where, fmt.Sprintf(`
				NOT EXISTS (
					SELECT 1
					FROM report_submissions rs
					WHERE rs.plan_item_id = pi.id
					  AND rs.submitted_by = %s
				)
			`, ph))
		}
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	err = h.db.QueryRow(`
		SELECT COUNT(*)
		FROM indicator_year_targets iyt
		JOIN planning_period_indicators ppi
		  ON ppi.id = iyt.indicator_id
		LEFT JOIN plan_items pi
		       ON pi.indicator_id = ppi.id
		      AND pi.year = iyt.year
		WHERE `+whereClause, args...).Scan(&total)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to count plan indicators"})
		return
	}

	queryArgs := append([]any{}, args...)
	offset := (page - 1) * limit
	queryArgs = append(queryArgs, limit, offset)

	rows, err := h.db.Query(`
		SELECT ppi.id,
		       ppi.target_indicator,
		       COALESCE(ppi.unit, ''),
		       COALESCE(iyt.planned_value, ''),
		       COALESCE(NULLIF(pi.development_indicator, ''), ppi.target_indicator),
		       COALESCE(pi.activities, ''),
		       COALESCE(TO_CHAR(pi.execution_start_date, 'YYYY-MM-DD'), ''),
		       COALESCE(TO_CHAR(pi.execution_end_date, 'YYYY-MM-DD'), ''),
		       CASE
		           WHEN COALESCE(TRIM(pi.activities), '') = ''
		                OR pi.execution_start_date IS NULL
		                OR pi.execution_end_date IS NULL
		                OR NOT EXISTS (
		                    SELECT 1
		                    FROM plan_item_responsibles res_chk
		                    WHERE res_chk.plan_item_id = pi.id
		                )
		           THEN 'not_filled'
		           WHEN CURRENT_DATE < pi.execution_start_date THEN 'upcoming'
		           WHEN CURRENT_DATE > pi.execution_end_date THEN 'overdue'
		           ELSE 'in_progress'
		       END,
		       CASE
		           WHEN pi.execution_start_date IS NOT NULL AND pi.execution_end_date IS NOT NULL
		           THEN TO_CHAR(pi.execution_start_date, 'DD.MM.YYYY') || ' - ' || TO_CHAR(pi.execution_end_date, 'DD.MM.YYYY')
		           ELSE ''
		       END,
		       COALESCE((
		           SELECT CASE
		               WHEN COUNT(*) FILTER (WHERE LOWER(TRIM(COALESCE(rs.status, ''))) IN ('completed', 'approved')) > 0
		               THEN 'completed'
		               WHEN COUNT(*) FILTER (WHERE LOWER(TRIM(COALESCE(rs.status, ''))) = 'rejected') > 0
		               THEN 'rejected'
		               WHEN COUNT(*) > 0
		               THEN 'pending'
		               ELSE ''
		           END
		           FROM report_submissions rs
		           WHERE rs.plan_item_id = pi.id
		       ), ''),
		       COALESCE((
		           SELECT TO_CHAR(
		               COALESCE(MAX(rs.submitted_at), MAX(rs.updated_at), MAX(rs.created_at)),
		               'YYYY-MM-DD"T"HH24:MI:SSTZH:TZM'
		           )
		           FROM report_submissions rs
		           WHERE rs.plan_item_id = pi.id
		       ), ''),
		       COALESCE((
		           SELECT jsonb_agg(res.user_id ORDER BY res.user_id)
		           FROM plan_item_responsibles res
		           WHERE res.plan_item_id = pi.id
		       ), '[]'::jsonb)
		FROM indicator_year_targets iyt
		JOIN planning_period_indicators ppi
		  ON ppi.id = iyt.indicator_id
		LEFT JOIN plan_items pi
		       ON pi.indicator_id = ppi.id
		      AND pi.year = iyt.year
		WHERE `+whereClause+`
		ORDER BY ppi.id ASC
		LIMIT $`+strconv.Itoa(len(queryArgs)-1)+`
		OFFSET $`+strconv.Itoa(len(queryArgs)), queryArgs...)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to load plan indicators"})
		return
	}
	defer rows.Close()

	items := make([]planIndicatorRow, 0)
	for rows.Next() {
		var item planIndicatorRow
		var responsibleUserIDsRaw []byte
		if err := rows.Scan(
			&item.IndicatorID,
			&item.SourceIndicator,
			&item.Unit,
			&item.PlannedValue,
			&item.DevelopmentIndicator,
			&item.Activities,
			&item.ExecutionStartDate,
			&item.ExecutionEndDate,
			&item.ScheduleStatus,
			&item.ExecutionDeadline,
			&item.ReportStatus,
			&item.LastSubmittedAt,
			&responsibleUserIDsRaw,
		); err != nil {
			c.JSON(500, errorResponse{Error: "failed to parse plan indicators"})
			return
		}

		parsedIDs, parseErr := parseResponsibleUserIDs(responsibleUserIDsRaw)
		if parseErr != nil {
			c.JSON(500, errorResponse{Error: "failed to parse responsible users"})
			return
		}
		item.ResponsibleUserIDs = parsedIDs
		item.MeasurementUnit = item.Unit
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to iterate plan indicators"})
		return
	}

	if err := h.populateResponsibleNames(items); err != nil {
		c.JSON(500, errorResponse{Error: "failed to resolve responsible users"})
		return
	}

	totalPages := total / limit
	if total%limit != 0 {
		totalPages++
	}

	c.JSON(200, planIndicatorListResponse{
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

// upsertPlanIndicator godoc
// @Summary Save plan row details for selected indicator/year
// @Description Creates or updates editable plan fields in normalized tables
// @Tags plans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param indicator_id path int true "Planning period indicator ID"
// @Param year query int true "Year"
// @Param payload body upsertPlanIndicatorRequest true "Editable plan row payload"
// @Success 200 {object} planIndicatorRow
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /plans/indicators/{indicator_id} [put]
func (h *Handler) upsertPlanIndicator(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(401, errorResponse{Error: "unauthorized"})
		return
	}
	if user.Role != "admin" {
		c.JSON(403, errorResponse{Error: "forbidden"})
		return
	}

	indicatorID, err := strconv.Atoi(c.Param("indicator_id"))
	if err != nil {
		c.JSON(400, errorResponse{Error: "invalid indicator id"})
		return
	}

	year, err := resolveCurrentPlanYear(c.Query("year"))
	if err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	var req upsertPlanIndicatorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	executionStartDate, executionEndDate, err := parseExecutionDateRange(req.ExecutionStartDate, req.ExecutionEndDate)
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
	if err := h.ensureIndicatorYearExists(indicatorID, strconv.Itoa(year)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(404, errorResponse{Error: "indicator for selected year not found"})
			return
		}
		c.JSON(500, errorResponse{Error: "failed to validate indicator"})
		return
	}

	responsibleUserIDs := normalizeResponsibleUserIDs(req.ResponsibleUserIDs)
	responsibleByID, err := h.resolveProrectorNames(responsibleUserIDs)
	if err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}
	for _, id := range responsibleUserIDs {
		if _, ok := responsibleByID[id]; !ok {
			c.JSON(400, errorResponse{Error: "invalid responsible users selection"})
			return
		}
	}

	tx, err := h.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to start transaction"})
		return
	}
	defer tx.Rollback()

	var planItemID int64
	err = tx.QueryRow(`
		INSERT INTO plan_items (
			indicator_id,
			year,
			development_indicator,
			activities,
			execution_deadline,
			execution_start_date,
			execution_end_date,
			status,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, '', $5, $6, 'draft', NOW(), NOW())
		ON CONFLICT (indicator_id, year)
		DO UPDATE SET
			development_indicator = EXCLUDED.development_indicator,
			activities = EXCLUDED.activities,
			execution_deadline = '',
			execution_start_date = EXCLUDED.execution_start_date,
			execution_end_date = EXCLUDED.execution_end_date,
			updated_at = NOW()
		RETURNING id
	`, indicatorID, year, strings.TrimSpace(req.DevelopmentIndicator), strings.TrimSpace(req.Activities), executionStartDate, executionEndDate).Scan(&planItemID)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to save plan item"})
		return
	}

	if _, err := tx.Exec(`
		DELETE FROM plan_item_responsibles
		WHERE plan_item_id = $1
	`, planItemID); err != nil {
		c.JSON(500, errorResponse{Error: "failed to reset responsible users"})
		return
	}

	for _, responsibleUserID := range responsibleUserIDs {
		if _, err := tx.Exec(`
			INSERT INTO plan_item_responsibles (plan_item_id, user_id, created_at)
			VALUES ($1, $2, NOW())
			ON CONFLICT (plan_item_id, user_id) DO NOTHING
		`, planItemID, responsibleUserID); err != nil {
			c.JSON(500, errorResponse{Error: "failed to save responsible users"})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to commit plan item"})
		return
	}

	item, err := h.fetchPlanIndicatorRow(indicatorID, year, strconv.Itoa(year))
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to load saved plan indicator"})
		return
	}

	c.JSON(200, item)
}

func parseYear(raw string) (int, string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, "", fmt.Errorf("year is required")
	}

	year, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, "", fmt.Errorf("year must be integer")
	}
	if year < 2000 || year > 2100 {
		return 0, "", fmt.Errorf("year is out of supported range")
	}

	return year, strconv.Itoa(year), nil
}

func currentPlanYear() int {
	return time.Now().Year()
}

func resolveCurrentPlanYear(raw string) (int, error) {
	currentYear := currentPlanYear()
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return currentYear, nil
	}

	year, _, err := parseYear(trimmed)
	if err != nil {
		return 0, err
	}
	if year != currentYear {
		return 0, fmt.Errorf("only current year is allowed: %d", currentYear)
	}

	return year, nil
}

func parseExecutionDateRange(startRaw, endRaw string) (time.Time, time.Time, error) {
	startValue := strings.TrimSpace(startRaw)
	endValue := strings.TrimSpace(endRaw)
	if startValue == "" || endValue == "" {
		return time.Time{}, time.Time{}, fmt.Errorf("execution_start_date and execution_end_date are required")
	}

	startDate, err := time.Parse("2006-01-02", startValue)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("execution_start_date must be YYYY-MM-DD")
	}
	endDate, err := time.Parse("2006-01-02", endValue)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("execution_end_date must be YYYY-MM-DD")
	}
	if startDate.After(endDate) {
		return time.Time{}, time.Time{}, fmt.Errorf("execution_start_date cannot be later than execution_end_date")
	}

	return startDate, endDate, nil
}

func (h *Handler) ensurePlanningPeriodTable() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS planning_period_indicators (
			id BIGSERIAL PRIMARY KEY,
			target_indicator TEXT NOT NULL,
			unit VARCHAR(32) NOT NULL,
			year_values JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS indicator_year_targets (
			id BIGSERIAL PRIMARY KEY,
			indicator_id BIGINT NOT NULL REFERENCES planning_period_indicators(id) ON DELETE CASCADE,
			year INT NOT NULL CHECK (year >= 2000 AND year <= 2100),
			planned_value TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (indicator_id, year)
		);`,
		`DO $$
		BEGIN
		  IF EXISTS (
		    SELECT 1
		    FROM information_schema.columns
		    WHERE table_schema = 'public'
		      AND table_name = 'indicator_year_targets'
		      AND column_name = 'planned_value'
		      AND data_type <> 'text'
		  ) THEN
		    ALTER TABLE indicator_year_targets
		      ALTER COLUMN planned_value TYPE TEXT
		      USING TRIM(BOTH FROM planned_value::TEXT);
		  END IF;
		END $$;`,
		`CREATE INDEX IF NOT EXISTS indicator_year_targets_year_idx ON indicator_year_targets (year);`,
		`INSERT INTO indicator_year_targets (indicator_id, year, planned_value, created_at, updated_at)
		SELECT ppi.id,
		       (kv.key)::INT,
		       kv.value,
		       NOW(),
		       NOW()
		FROM planning_period_indicators ppi
		CROSS JOIN LATERAL jsonb_each_text(COALESCE(ppi.year_values, '{}'::jsonb)) kv
		WHERE kv.key ~ '^[0-9]{4}$'
		ON CONFLICT (indicator_id, year)
		DO UPDATE SET planned_value = EXCLUDED.planned_value,
		              updated_at = NOW();`,
	}

	for _, stmt := range statements {
		if _, err := h.db.Exec(stmt); err != nil {
			return err
		}
	}

	return nil
}

func (h *Handler) ensurePlanIndicatorDetailsTable() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS plan_items (
			id BIGSERIAL PRIMARY KEY,
			indicator_id BIGINT NOT NULL REFERENCES planning_period_indicators(id) ON DELETE CASCADE,
			year INT NOT NULL CHECK (year >= 2000 AND year <= 2100),
			development_indicator TEXT NOT NULL DEFAULT '',
			activities TEXT NOT NULL DEFAULT '',
			execution_deadline TEXT NOT NULL DEFAULT '',
			execution_start_date DATE NULL,
			execution_end_date DATE NULL,
			status VARCHAR(32) NOT NULL DEFAULT 'draft',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (indicator_id, year)
		);`,
		`ALTER TABLE plan_items
		  ADD COLUMN IF NOT EXISTS execution_start_date DATE NULL;`,
		`ALTER TABLE plan_items
		  ADD COLUMN IF NOT EXISTS execution_end_date DATE NULL;`,
		`CREATE INDEX IF NOT EXISTS plan_items_year_idx ON plan_items (year);`,
		`CREATE INDEX IF NOT EXISTS plan_items_status_idx ON plan_items (status);`,
		`CREATE TABLE IF NOT EXISTS plan_item_responsibles (
			plan_item_id BIGINT NOT NULL REFERENCES plan_items(id) ON DELETE CASCADE,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (plan_item_id, user_id)
		);`,
		`CREATE INDEX IF NOT EXISTS plan_item_responsibles_user_idx ON plan_item_responsibles (user_id);`,
		`INSERT INTO plan_items (
			indicator_id,
			year,
			development_indicator,
			activities,
			execution_deadline,
			execution_start_date,
			execution_end_date,
			status,
			created_at,
			updated_at
		)
		SELECT pid.planning_period_indicator_id,
		       pid.year,
		       COALESCE(pid.development_indicator, ''),
		       COALESCE(pid.activities, ''),
		       '',
		       NULL,
		       NULL,
		       'draft',
		       COALESCE(pid.created_at, NOW()),
		       COALESCE(pid.updated_at, NOW())
		FROM plan_indicator_details pid
		ON CONFLICT (indicator_id, year)
		DO UPDATE SET
			development_indicator = EXCLUDED.development_indicator,
			activities = EXCLUDED.activities,
			execution_deadline = EXCLUDED.execution_deadline,
			updated_at = NOW();`,
		`INSERT INTO plan_items (
			indicator_id,
			year,
			development_indicator,
			activities,
			execution_deadline,
			execution_start_date,
			execution_end_date,
			status,
			created_at,
			updated_at
		)
		SELECT pir.planning_period_indicator_id,
		       pir.year,
		       COALESCE(NULLIF(TRIM(pid.development_indicator), ''), ppi.target_indicator, ''),
		       COALESCE(pid.activities, ''),
		       '',
		       NULL,
		       NULL,
		       'draft',
		       COALESCE(pir.created_at, NOW()),
		       COALESCE(pir.updated_at, NOW())
		FROM plan_indicator_reports pir
		LEFT JOIN plan_indicator_details pid
		       ON pid.planning_period_indicator_id = pir.planning_period_indicator_id
		      AND pid.year = pir.year
		LEFT JOIN planning_period_indicators ppi
		       ON ppi.id = pir.planning_period_indicator_id
		ON CONFLICT (indicator_id, year)
		DO NOTHING;`,
		`INSERT INTO plan_item_responsibles (plan_item_id, user_id, created_at)
		SELECT pi.id,
		       elem.value::BIGINT,
		       NOW()
		FROM plan_indicator_details pid
		JOIN plan_items pi
		  ON pi.indicator_id = pid.planning_period_indicator_id
		 AND pi.year = pid.year
		CROSS JOIN LATERAL jsonb_array_elements_text(COALESCE(pid.responsible_user_ids, '[]'::jsonb)) elem(value)
		WHERE elem.value ~ '^[0-9]+$'
		ON CONFLICT (plan_item_id, user_id) DO NOTHING;`,
		`INSERT INTO plan_item_responsibles (plan_item_id, user_id, created_at)
		SELECT pi.id,
		       pir.submitted_by,
		       COALESCE(pir.created_at, NOW())
		FROM plan_indicator_reports pir
		JOIN users u
		  ON u.id = pir.submitted_by
		 AND u.role = 'prorector'
		JOIN plan_items pi
		  ON pi.indicator_id = pir.planning_period_indicator_id
		 AND pi.year = pir.year
		ON CONFLICT (plan_item_id, user_id) DO NOTHING;`,
	}

	for _, stmt := range statements {
		if _, err := h.db.Exec(stmt); err != nil {
			return err
		}
	}

	if err := h.cleanupLegacyPlanDataOnce(); err != nil {
		return err
	}
	return nil
}

func (h *Handler) ensureIndicatorYearExists(indicatorID int, yearKey string) error {
	year, err := strconv.Atoi(yearKey)
	if err != nil {
		return fmt.Errorf("year key must be numeric")
	}

	var id int64
	err = h.db.QueryRow(`
		SELECT indicator_id
		FROM indicator_year_targets
		WHERE indicator_id = $1
		  AND year = $2
		LIMIT 1
	`, indicatorID, year).Scan(&id)
	if err != nil {
		return err
	}
	return nil
}

func (h *Handler) fetchPlanIndicatorRow(indicatorID int, year int, yearKey string) (planIndicatorRow, error) {
	var item planIndicatorRow
	var responsibleUserIDsRaw []byte
	err := h.db.QueryRow(`
		SELECT ppi.id,
		       ppi.target_indicator,
		       COALESCE(ppi.unit, ''),
		       COALESCE(iyt.planned_value, ''),
		       COALESCE(NULLIF(pi.development_indicator, ''), ppi.target_indicator),
		       COALESCE(pi.activities, ''),
		       COALESCE(TO_CHAR(pi.execution_start_date, 'YYYY-MM-DD'), ''),
		       COALESCE(TO_CHAR(pi.execution_end_date, 'YYYY-MM-DD'), ''),
		       CASE
		           WHEN COALESCE(TRIM(pi.activities), '') = ''
		                OR pi.execution_start_date IS NULL
		                OR pi.execution_end_date IS NULL
		                OR NOT EXISTS (
		                    SELECT 1
		                    FROM plan_item_responsibles res_chk
		                    WHERE res_chk.plan_item_id = pi.id
		                )
		           THEN 'not_filled'
		           WHEN CURRENT_DATE < pi.execution_start_date THEN 'upcoming'
		           WHEN CURRENT_DATE > pi.execution_end_date THEN 'overdue'
		           ELSE 'in_progress'
		       END,
		       CASE
		           WHEN pi.execution_start_date IS NOT NULL AND pi.execution_end_date IS NOT NULL
		           THEN TO_CHAR(pi.execution_start_date, 'DD.MM.YYYY') || ' - ' || TO_CHAR(pi.execution_end_date, 'DD.MM.YYYY')
		           ELSE ''
		       END,
		       COALESCE((
		           SELECT jsonb_agg(res.user_id ORDER BY res.user_id)
		           FROM plan_item_responsibles res
		           WHERE res.plan_item_id = pi.id
		       ), '[]'::jsonb)
		FROM planning_period_indicators ppi
		JOIN indicator_year_targets iyt
		  ON iyt.indicator_id = ppi.id
		 AND iyt.year = $1
		LEFT JOIN plan_items pi
		       ON pi.indicator_id = ppi.id
		      AND pi.year = iyt.year
		WHERE ppi.id = $2
		LIMIT 1
	`, year, indicatorID).Scan(
		&item.IndicatorID,
		&item.SourceIndicator,
		&item.Unit,
		&item.PlannedValue,
		&item.DevelopmentIndicator,
		&item.Activities,
		&item.ExecutionStartDate,
		&item.ExecutionEndDate,
		&item.ScheduleStatus,
		&item.ExecutionDeadline,
		&responsibleUserIDsRaw,
	)
	if err != nil {
		return planIndicatorRow{}, err
	}

	parsedIDs, err := parseResponsibleUserIDs(responsibleUserIDsRaw)
	if err != nil {
		return planIndicatorRow{}, err
	}
	item.ResponsibleUserIDs = parsedIDs
	item.MeasurementUnit = item.Unit

	rows := []planIndicatorRow{item}
	if err := h.populateResponsibleNames(rows); err != nil {
		return planIndicatorRow{}, err
	}
	item = rows[0]

	return item, nil
}

func (h *Handler) cleanupLegacyPlanDataOnce() error {
	if _, err := h.db.Exec(`
		CREATE TABLE IF NOT EXISTS system_maintenance_flags (
			key TEXT PRIMARY KEY,
			done_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`); err != nil {
		return err
	}

	var alreadyDone bool
	if err := h.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1
			FROM system_maintenance_flags
			WHERE key = 'plans_date_range_reset_v1'
		)
	`).Scan(&alreadyDone); err != nil {
		return err
	}
	if alreadyDone {
		return nil
	}

	tx, err := h.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	tablesToClear := []string{
		"report_status_history",
		"report_files",
		"report_submissions",
		"plan_item_responsibles",
		"plan_items",
		"plan_indicator_report_files",
		"plan_indicator_reports",
		"plan_indicator_details",
	}
	for _, tableName := range tablesToClear {
		var exists bool
		if err := tx.QueryRow(`SELECT to_regclass($1) IS NOT NULL`, tableName).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			continue
		}
		if _, err := tx.Exec(fmt.Sprintf(`TRUNCATE TABLE %s RESTART IDENTITY CASCADE`, tableName)); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`
		INSERT INTO system_maintenance_flags (key, done_at)
		VALUES ('plans_date_range_reset_v1', NOW())
		ON CONFLICT (key) DO NOTHING
	`); err != nil {
		return err
	}

	return tx.Commit()
}

func parseResponsibleUserIDs(raw []byte) ([]int64, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return []int64{}, nil
	}

	var values []int64
	if err := json.Unmarshal(trimmed, &values); err == nil {
		return normalizeResponsibleUserIDs(values), nil
	}

	var generic []any
	if err := json.Unmarshal(trimmed, &generic); err != nil {
		return nil, err
	}

	parsed := make([]int64, 0, len(generic))
	for _, value := range generic {
		switch typed := value.(type) {
		case float64:
			parsed = append(parsed, int64(typed))
		case string:
			id, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
			if err != nil {
				return nil, err
			}
			parsed = append(parsed, id)
		default:
			return nil, fmt.Errorf("unsupported responsible user id type: %T", value)
		}
	}

	return normalizeResponsibleUserIDs(parsed), nil
}

func normalizeResponsibleUserIDs(ids []int64) []int64 {
	if len(ids) == 0 {
		return []int64{}
	}

	seen := make(map[int64]struct{}, len(ids))
	normalized := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}

	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i] < normalized[j]
	})
	return normalized
}

func (h *Handler) resolveProrectorNames(ids []int64) (map[int64]string, error) {
	if len(ids) == 0 {
		return map[int64]string{}, nil
	}

	query, args := buildUserLookupQuery(ids, true)
	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to validate responsible users")
	}
	defer rows.Close()

	result := make(map[int64]string, len(ids))
	for rows.Next() {
		var id int64
		var fullName string
		if err := rows.Scan(&id, &fullName); err != nil {
			return nil, fmt.Errorf("failed to parse responsible users")
		}
		result[id] = strings.TrimSpace(fullName)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate responsible users")
	}

	if len(result) != len(ids) {
		missing := make([]string, 0)
		for _, id := range ids {
			if _, ok := result[id]; !ok {
				missing = append(missing, strconv.FormatInt(id, 10))
			}
		}
		return nil, fmt.Errorf("invalid prorector ids: %s", strings.Join(missing, ", "))
	}

	return result, nil
}

func (h *Handler) populateResponsibleNames(items []planIndicatorRow) error {
	idSet := make(map[int64]struct{})
	for _, item := range items {
		for _, id := range item.ResponsibleUserIDs {
			idSet[id] = struct{}{}
		}
	}
	if len(idSet) == 0 {
		return nil
	}

	ids := make([]int64, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	query, args := buildUserLookupQuery(ids, false)
	rows, err := h.db.Query(query, args...)
	if err != nil {
		return fmt.Errorf("failed to load responsible users")
	}
	defer rows.Close()

	nameByID := make(map[int64]string, len(ids))
	for rows.Next() {
		var id int64
		var fullName string
		if err := rows.Scan(&id, &fullName); err != nil {
			return fmt.Errorf("failed to parse responsible users")
		}
		nameByID[id] = strings.TrimSpace(fullName)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to iterate responsible users")
	}

	for i := range items {
		if len(items[i].ResponsibleUserIDs) == 0 {
			items[i].Responsible = ""
			continue
		}

		names := make([]string, 0, len(items[i].ResponsibleUserIDs))
		for _, id := range items[i].ResponsibleUserIDs {
			if name, ok := nameByID[id]; ok && name != "" {
				names = append(names, name)
			}
		}
		items[i].Responsible = strings.Join(names, ", ")
	}

	return nil
}

func buildUserLookupQuery(ids []int64, onlyProrector bool) (string, []any) {
	placeholders := make([]string, 0, len(ids))
	args := make([]any, 0, len(ids))
	for i, id := range ids {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
		args = append(args, id)
	}

	whereRole := ""
	if onlyProrector {
		whereRole = "AND role = 'prorector'"
	}

	query := fmt.Sprintf(`
		SELECT id, COALESCE(NULLIF(TRIM(full_name), ''), username)
		FROM users
		WHERE id IN (%s)
		  %s
	`, strings.Join(placeholders, ","), whereRole)

	return query, args
}

func containsInt64(values []int64, wanted int64) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

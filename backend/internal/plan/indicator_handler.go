package plan

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"OperationPlan/internal/middleware"

	"github.com/gin-gonic/gin"
)

type planYearsResponse struct {
	Years []int `json:"years"`
}

type planIndicatorListResponse struct {
	Year  int                `json:"year"`
	Items []planIndicatorRow `json:"items"`
}

type planIndicatorRow struct {
	IndicatorID          int    `json:"indicator_id"`
	SourceIndicator      string `json:"source_indicator"`
	Unit                 string `json:"unit"`
	PlannedValue         string `json:"planned_value"`
	DevelopmentIndicator string `json:"development_indicator"`
	Activities           string `json:"activities"`
	ExecutionDeadline    string `json:"execution_deadline"`
	Responsible          string `json:"responsible"`
}

type upsertPlanIndicatorRequest struct {
	DevelopmentIndicator string `json:"development_indicator"`
	Activities           string `json:"activities"`
	ExecutionDeadline    string `json:"execution_deadline"`
	Responsible          string `json:"responsible"`
}

// listPlanYears godoc
// @Summary List available plan years
// @Description Returns years extracted from planning period indicators
// @Tags plans
// @Produce json
// @Security BearerAuth
// @Success 200 {object} planYearsResponse
// @Failure 500 {object} errorResponse
// @Router /plans/years [get]
func (h *Handler) listPlanYears(c *gin.Context) {
	if err := h.ensurePlanningPeriodTable(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to prepare planning period storage"})
		return
	}

	rows, err := h.db.Query(`SELECT year_values FROM planning_period_indicators`)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to load planning years"})
		return
	}
	defer rows.Close()

	yearSet := make(map[int]struct{})
	for rows.Next() {
		var raw any
		if err := rows.Scan(&raw); err != nil {
			c.JSON(500, errorResponse{Error: "failed to parse planning years"})
			return
		}

		values, parseErr := parseYearValues(raw)
		if parseErr != nil {
			c.JSON(500, errorResponse{Error: "failed to parse planning years"})
			return
		}

		for key := range values {
			year, convErr := strconv.Atoi(key)
			if convErr != nil {
				continue
			}
			yearSet[year] = struct{}{}
		}
	}

	if err := rows.Err(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to iterate planning years"})
		return
	}

	years := make([]int, 0, len(yearSet))
	for year := range yearSet {
		years = append(years, year)
	}
	sort.Ints(years)

	c.JSON(200, planYearsResponse{Years: years})
}

// listPlanIndicators godoc
// @Summary List plan indicators for selected year
// @Description Returns plan rows based on planning period indicators for selected year
// @Tags plans
// @Produce json
// @Security BearerAuth
// @Param year query int true "Year"
// @Success 200 {object} planIndicatorListResponse
// @Failure 400 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /plans/indicators [get]
func (h *Handler) listPlanIndicators(c *gin.Context) {
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

	rows, err := h.db.Query(`
		SELECT ppi.id,
		       ppi.target_indicator,
		       ppi.unit,
		       COALESCE(ppi.year_values ->> $1, ''),
		       COALESCE(NULLIF(pid.development_indicator, ''), ppi.target_indicator),
		       COALESCE(pid.activities, ''),
		       COALESCE(pid.execution_deadline, ''),
		       COALESCE(pid.responsible, '')
		FROM planning_period_indicators ppi
		LEFT JOIN plan_indicator_details pid
		       ON pid.planning_period_indicator_id = ppi.id
		      AND pid.year = $2
		WHERE ppi.year_values ? $1
		ORDER BY ppi.id ASC
	`, yearKey, year)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to load plan indicators"})
		return
	}
	defer rows.Close()

	items := make([]planIndicatorRow, 0)
	for rows.Next() {
		var item planIndicatorRow
		if err := rows.Scan(
			&item.IndicatorID,
			&item.SourceIndicator,
			&item.Unit,
			&item.PlannedValue,
			&item.DevelopmentIndicator,
			&item.Activities,
			&item.ExecutionDeadline,
			&item.Responsible,
		); err != nil {
			c.JSON(500, errorResponse{Error: "failed to parse plan indicators"})
			return
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to iterate plan indicators"})
		return
	}

	c.JSON(200, planIndicatorListResponse{
		Year:  year,
		Items: items,
	})
}

// upsertPlanIndicator godoc
// @Summary Save plan row details for selected indicator/year
// @Description Creates or updates editable fields in plans table
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

	year, yearKey, err := parseYear(c.Query("year"))
	if err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	var req upsertPlanIndicatorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
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

	if err := h.ensureIndicatorYearExists(indicatorID, yearKey); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(404, errorResponse{Error: "indicator for selected year not found"})
			return
		}
		c.JSON(500, errorResponse{Error: "failed to validate indicator"})
		return
	}

	_, err = h.db.Exec(`
		INSERT INTO plan_indicator_details (
			planning_period_indicator_id,
			year,
			development_indicator,
			activities,
			execution_deadline,
			responsible,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		ON CONFLICT (planning_period_indicator_id, year)
		DO UPDATE SET
			development_indicator = EXCLUDED.development_indicator,
			activities = EXCLUDED.activities,
			execution_deadline = EXCLUDED.execution_deadline,
			responsible = EXCLUDED.responsible,
			updated_at = NOW()
	`,
		indicatorID,
		year,
		strings.TrimSpace(req.DevelopmentIndicator),
		strings.TrimSpace(req.Activities),
		strings.TrimSpace(req.ExecutionDeadline),
		strings.TrimSpace(req.Responsible),
	)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to save plan indicator"})
		return
	}

	item, err := h.fetchPlanIndicatorRow(indicatorID, year, yearKey)
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

func parseYearValues(raw any) (map[string]float64, error) {
	if raw == nil {
		return map[string]float64{}, nil
	}

	var data []byte
	switch typed := raw.(type) {
	case []byte:
		data = typed
	case string:
		data = []byte(typed)
	default:
		return nil, fmt.Errorf("unsupported year_values type: %T", raw)
	}

	if len(data) == 0 {
		return map[string]float64{}, nil
	}

	var generic map[string]any
	if err := json.Unmarshal(data, &generic); err != nil {
		return nil, err
	}

	values := make(map[string]float64, len(generic))
	for key, value := range generic {
		number, err := parseNumericValue(value)
		if err != nil {
			return nil, fmt.Errorf("invalid value for year %s: %w", key, err)
		}
		values[key] = number
	}

	return values, nil
}

func parseNumericValue(value any) (float64, error) {
	switch typed := value.(type) {
	case float64:
		return typed, nil
	case float32:
		return float64(typed), nil
	case int:
		return float64(typed), nil
	case int64:
		return float64(typed), nil
	case json.Number:
		return typed.Float64()
	case string:
		normalized := strings.TrimSpace(strings.ReplaceAll(typed, ",", "."))
		if normalized == "" {
			return 0, fmt.Errorf("empty string")
		}
		return strconv.ParseFloat(normalized, 64)
	default:
		return 0, fmt.Errorf("unsupported type %T", value)
	}
}

func (h *Handler) ensurePlanningPeriodTable() error {
	_, err := h.db.Exec(`
		CREATE TABLE IF NOT EXISTS planning_period_indicators (
			id BIGSERIAL PRIMARY KEY,
			target_indicator TEXT NOT NULL,
			unit VARCHAR(32) NOT NULL,
			year_values JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	return err
}

func (h *Handler) ensurePlanIndicatorDetailsTable() error {
	_, err := h.db.Exec(`
		CREATE TABLE IF NOT EXISTS plan_indicator_details (
			id BIGSERIAL PRIMARY KEY,
			planning_period_indicator_id BIGINT NOT NULL REFERENCES planning_period_indicators(id) ON DELETE CASCADE,
			year INT NOT NULL,
			development_indicator TEXT NOT NULL DEFAULT '',
			activities TEXT NOT NULL DEFAULT '',
			execution_deadline TEXT NOT NULL DEFAULT '',
			responsible TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (planning_period_indicator_id, year)
		);
	`)
	return err
}

func (h *Handler) ensureIndicatorYearExists(indicatorID int, yearKey string) error {
	var id int
	err := h.db.QueryRow(`
		SELECT id
		FROM planning_period_indicators
		WHERE id = $1
		  AND year_values ? $2
	`, indicatorID, yearKey).Scan(&id)
	if err != nil {
		return err
	}
	return nil
}

func (h *Handler) fetchPlanIndicatorRow(indicatorID int, year int, yearKey string) (planIndicatorRow, error) {
	var item planIndicatorRow
	err := h.db.QueryRow(`
		SELECT ppi.id,
		       ppi.target_indicator,
		       ppi.unit,
		       COALESCE(ppi.year_values ->> $1, ''),
		       COALESCE(NULLIF(pid.development_indicator, ''), ppi.target_indicator),
		       COALESCE(pid.activities, ''),
		       COALESCE(pid.execution_deadline, ''),
		       COALESCE(pid.responsible, '')
		FROM planning_period_indicators ppi
		LEFT JOIN plan_indicator_details pid
		       ON pid.planning_period_indicator_id = ppi.id
		      AND pid.year = $2
		WHERE ppi.id = $3
		  AND ppi.year_values ? $1
		LIMIT 1
	`, yearKey, year, indicatorID).Scan(
		&item.IndicatorID,
		&item.SourceIndicator,
		&item.Unit,
		&item.PlannedValue,
		&item.DevelopmentIndicator,
		&item.Activities,
		&item.ExecutionDeadline,
		&item.Responsible,
	)
	if err != nil {
		return planIndicatorRow{}, err
	}

	return item, nil
}

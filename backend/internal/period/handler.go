package period

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"

	"OperationPlan/internal/middleware"
	"OperationPlan/internal/model"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	db *sql.DB
}

type createPlanningPeriodRequest struct {
	TargetIndicator string             `json:"target_indicator" binding:"required"`
	Unit            string             `json:"unit" binding:"required"`
	YearValues      map[string]float64 `json:"year_values" binding:"required"`
}

type updatePlanningPeriodRequest struct {
	TargetIndicator *string            `json:"target_indicator"`
	Unit            *string            `json:"unit"`
	YearValues      map[string]float64 `json:"year_values"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type planningPeriodListResponse struct {
	Items []model.PlanningPeriodIndicator `json:"items"`
}

type rowScanner interface {
	Scan(dest ...any) error
}

func RegisterRoutes(router gin.IRoutes, db *sql.DB) {
	h := &Handler{db: db}

	router.GET("/planning-period", h.list)
	router.POST("/planning-period", h.create)
	router.PATCH("/planning-period/:id", h.update)
}

// list godoc
// @Summary List planning period indicators
// @Description Returns admin table "Плановый период по годам"
// @Tags planning-period
// @Produce json
// @Security BearerAuth
// @Success 200 {object} planningPeriodListResponse
// @Failure 500 {object} errorResponse
// @Router /planning-period [get]
func (h *Handler) list(c *gin.Context) {
	if err := h.ensurePlanningPeriodTable(); err != nil {
		slog.Error("failed to ensure planning_period_indicators table", "error", err.Error())
		c.JSON(500, errorResponse{Error: "failed to prepare planning period storage"})
		return
	}

	rows, err := h.db.Query(`
		SELECT id, target_indicator, unit, year_values, created_at, updated_at
		FROM planning_period_indicators
		ORDER BY id ASC
	`)
	if err != nil {
		slog.Error("failed to load planning period data", "error", err.Error())
		c.JSON(500, errorResponse{Error: "failed to load planning period data"})
		return
	}
	defer rows.Close()

	items := make([]model.PlanningPeriodIndicator, 0)
	for rows.Next() {
		item, scanErr := scanIndicator(rows)
		if scanErr != nil {
			slog.Error("failed to scan planning period row", "error", scanErr.Error())
			c.JSON(500, errorResponse{Error: "failed to parse planning period data"})
			return
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		slog.Error("failed to iterate planning period rows", "error", err.Error())
		c.JSON(500, errorResponse{Error: "failed to iterate planning period data"})
		return
	}

	c.JSON(200, planningPeriodListResponse{Items: items})
}

// create godoc
// @Summary Create planning period indicator
// @Description Creates a new indicator row for "Плановый период по годам"
// @Tags planning-period
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param payload body createPlanningPeriodRequest true "Planning period indicator payload"
// @Success 201 {object} model.PlanningPeriodIndicator
// @Failure 400 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /planning-period [post]
func (h *Handler) create(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(401, errorResponse{Error: "unauthorized"})
		return
	}
	if user.Role != "admin" {
		c.JSON(403, errorResponse{Error: "forbidden"})
		return
	}

	if err := h.ensurePlanningPeriodTable(); err != nil {
		slog.Error("failed to ensure planning_period_indicators table", "error", err.Error())
		c.JSON(500, errorResponse{Error: "failed to prepare planning period storage"})
		return
	}

	var req createPlanningPeriodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	req.TargetIndicator = strings.TrimSpace(req.TargetIndicator)
	req.Unit = strings.TrimSpace(req.Unit)

	if req.TargetIndicator == "" || req.Unit == "" {
		c.JSON(400, errorResponse{Error: "target_indicator and unit are required"})
		return
	}

	if err := validateYearValues(req.YearValues); err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	yearValuesJSON, err := json.Marshal(req.YearValues)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to serialize year values"})
		return
	}

	row := h.db.QueryRow(`
		INSERT INTO planning_period_indicators (target_indicator, unit, year_values, created_at, updated_at)
		VALUES ($1, $2, $3::jsonb, NOW(), NOW())
		RETURNING id, target_indicator, unit, year_values, created_at, updated_at
	`, req.TargetIndicator, req.Unit, yearValuesJSON)

	item, scanErr := scanIndicator(row)
	if scanErr != nil {
		slog.Error("failed to create planning period indicator", "error", scanErr.Error())
		c.JSON(500, errorResponse{Error: "failed to create planning period indicator"})
		return
	}

	c.JSON(201, item)
}

// update godoc
// @Summary Update planning period indicator
// @Description Updates any parameter of an existing planning period indicator row
// @Tags planning-period
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Indicator ID"
// @Param payload body updatePlanningPeriodRequest true "Planning period indicator patch payload"
// @Success 200 {object} model.PlanningPeriodIndicator
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /planning-period/{id} [patch]
func (h *Handler) update(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		c.JSON(401, errorResponse{Error: "unauthorized"})
		return
	}
	if user.Role != "admin" {
		c.JSON(403, errorResponse{Error: "forbidden"})
		return
	}

	if err := h.ensurePlanningPeriodTable(); err != nil {
		slog.Error("failed to ensure planning_period_indicators table", "error", err.Error())
		c.JSON(500, errorResponse{Error: "failed to prepare planning period storage"})
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, errorResponse{Error: "invalid indicator id"})
		return
	}

	var req updatePlanningPeriodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	item, err := h.fetchIndicatorByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(404, errorResponse{Error: "indicator not found"})
			return
		}
		c.JSON(500, errorResponse{Error: "failed to load indicator"})
		return
	}

	if req.TargetIndicator != nil {
		value := strings.TrimSpace(*req.TargetIndicator)
		if value == "" {
			c.JSON(400, errorResponse{Error: "target_indicator cannot be empty"})
			return
		}
		item.TargetIndicator = value
	}

	if req.Unit != nil {
		value := strings.TrimSpace(*req.Unit)
		if value == "" {
			c.JSON(400, errorResponse{Error: "unit cannot be empty"})
			return
		}
		item.Unit = value
	}

	if req.YearValues != nil {
		if err := validateYearValues(req.YearValues); err != nil {
			c.JSON(400, errorResponse{Error: err.Error()})
			return
		}
		item.YearValues = req.YearValues
	}

	yearValuesJSON, err := json.Marshal(item.YearValues)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to serialize year values"})
		return
	}

	row := h.db.QueryRow(`
		UPDATE planning_period_indicators
		SET target_indicator = $1,
		    unit = $2,
		    year_values = $3::jsonb,
		    updated_at = NOW()
		WHERE id = $4
		RETURNING id, target_indicator, unit, year_values, created_at, updated_at
	`, item.TargetIndicator, item.Unit, yearValuesJSON, id)

	updated, scanErr := scanIndicator(row)
	if scanErr != nil {
		if errors.Is(scanErr, sql.ErrNoRows) {
			c.JSON(404, errorResponse{Error: "indicator not found"})
			return
		}
		c.JSON(500, errorResponse{Error: "failed to update indicator"})
		return
	}

	c.JSON(200, updated)
}

func (h *Handler) fetchIndicatorByID(id int) (model.PlanningPeriodIndicator, error) {
	row := h.db.QueryRow(`
		SELECT id, target_indicator, unit, year_values, created_at, updated_at
		FROM planning_period_indicators
		WHERE id = $1
	`, id)

	return scanIndicator(row)
}

func scanIndicator(scanner rowScanner) (model.PlanningPeriodIndicator, error) {
	var item model.PlanningPeriodIndicator
	var yearValuesRaw any

	err := scanner.Scan(
		&item.ID,
		&item.TargetIndicator,
		&item.Unit,
		&yearValuesRaw,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return model.PlanningPeriodIndicator{}, err
	}

	parsedValues, parseErr := parseYearValues(yearValuesRaw)
	if parseErr != nil {
		return model.PlanningPeriodIndicator{}, parseErr
	}

	if len(parsedValues) == 0 {
		item.YearValues = map[string]float64{}
		return item, nil
	}

	item.YearValues = parsedValues
	return item, nil
}

func parseYearValues(raw any) (map[string]float64, error) {
	if raw == nil {
		return map[string]float64{}, nil
	}

	var bytes []byte
	switch typed := raw.(type) {
	case []byte:
		bytes = typed
	case string:
		bytes = []byte(typed)
	default:
		return nil, fmt.Errorf("unsupported year_values type: %T", raw)
	}

	if len(bytes) == 0 {
		return map[string]float64{}, nil
	}

	var generic map[string]any
	if err := json.Unmarshal(bytes, &generic); err != nil {
		return nil, err
	}

	values := make(map[string]float64, len(generic))
	for key, value := range generic {
		number, err := coerceNumber(value)
		if err != nil {
			return nil, fmt.Errorf("invalid value for year %s: %w", key, err)
		}
		values[key] = number
	}

	return values, nil
}

func coerceNumber(value any) (float64, error) {
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
		parsed, err := strconv.ParseFloat(normalized, 64)
		if err != nil {
			return 0, err
		}
		return parsed, nil
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

func validateYearValues(values map[string]float64) error {
	if len(values) == 0 {
		return fmt.Errorf("year_values cannot be empty")
	}

	for key, value := range values {
		year, err := strconv.Atoi(strings.TrimSpace(key))
		if err != nil {
			return fmt.Errorf("invalid year key: %s", key)
		}

		if year < 2000 || year > 2100 {
			return fmt.Errorf("year out of supported range: %d", year)
		}

		if math.IsNaN(value) || math.IsInf(value, 0) {
			return fmt.Errorf("invalid numeric value for year %s", key)
		}
	}

	return nil
}

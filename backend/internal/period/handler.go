package period

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"OperationPlan/internal/middleware"
	"OperationPlan/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

type Handler struct {
	db *sql.DB
}

type createPlanningPeriodRequest struct {
	TargetIndicator string            `json:"target_indicator" binding:"required"`
	Unit            string            `json:"unit" binding:"required"`
	YearValues      map[string]string `json:"year_values" binding:"required"`
}

type updatePlanningPeriodRequest struct {
	TargetIndicator *string           `json:"target_indicator"`
	Unit            *string           `json:"unit"`
	YearValues      map[string]string `json:"year_values"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type listMeta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type planningPeriodListResponse struct {
	Items []model.PlanningPeriodIndicator `json:"items"`
	Meta  listMeta                        `json:"meta"`
}

type importPlanningPeriodResponse struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
	Skipped int `json:"skipped"`
}

func RegisterRoutes(router gin.IRoutes, db *sql.DB) {
	h := &Handler{db: db}

	router.GET("/planning-period", h.list)
	router.POST("/planning-period", h.create)
	router.PATCH("/planning-period/:id", h.update)
	router.POST("/planning-period/import", h.importExcel)
}

// list godoc
// @Summary List planning period indicators
// @Description Returns admin table "Плановый период по годам"
// @Tags planning-period
// @Produce json
// @Security BearerAuth
// @Param q query string false "Search text"
// @Param page query int false "Page number (default 1)"
// @Param limit query int false "Items per page (default 20, max 100)"
// @Success 200 {object} planningPeriodListResponse
// @Failure 500 {object} errorResponse
// @Router /planning-period [get]
func (h *Handler) list(c *gin.Context) {
	if err := h.ensurePlanningPeriodTable(); err != nil {
		slog.Error("failed to ensure planning period tables", "error", err.Error())
		c.JSON(500, errorResponse{Error: "failed to prepare planning period storage"})
		return
	}

	page, limit, err := parsePagination(c.Query("page"), c.Query("limit"))
	if err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}
	searchQuery := strings.TrimSpace(c.Query("q"))

	args := make([]any, 0, 2)
	where := []string{"1=1"}
	if searchQuery != "" {
		args = append(args, "%"+searchQuery+"%")
		placeholder := fmt.Sprintf("$%d", len(args))
		where = append(where, fmt.Sprintf("(target_indicator ILIKE %s OR unit ILIKE %s)", placeholder, placeholder))
	}
	whereClause := strings.Join(where, " AND ")

	var total int
	err = h.db.QueryRow(`
		SELECT COUNT(*)
		FROM planning_period_indicators
		WHERE `+whereClause, args...).Scan(&total)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to count planning period data"})
		return
	}

	queryArgs := append([]any{}, args...)
	offset := (page - 1) * limit
	queryArgs = append(queryArgs, limit, offset)
	rows, err := h.db.Query(`
		SELECT id, target_indicator, unit, created_at, updated_at
		FROM planning_period_indicators
		WHERE `+whereClause+`
		ORDER BY id ASC
		LIMIT $`+strconv.Itoa(len(queryArgs)-1)+`
		OFFSET $`+strconv.Itoa(len(queryArgs)), queryArgs...)
	if err != nil {
		slog.Error("failed to load planning period data", "error", err.Error())
		c.JSON(500, errorResponse{Error: "failed to load planning period data"})
		return
	}
	defer rows.Close()

	items := make([]model.PlanningPeriodIndicator, 0)
	ids := make([]int64, 0)
	for rows.Next() {
		var item model.PlanningPeriodIndicator
		if err := rows.Scan(
			&item.ID,
			&item.TargetIndicator,
			&item.Unit,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			c.JSON(500, errorResponse{Error: "failed to parse planning period data"})
			return
		}
		item.YearValues = map[string]string{}
		items = append(items, item)
		ids = append(ids, int64(item.ID))
	}
	if err := rows.Err(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to iterate planning period data"})
		return
	}

	valuesByIndicator, err := h.loadYearValuesByIndicatorIDs(ids)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to load yearly targets"})
		return
	}

	for i := range items {
		yearValues, ok := valuesByIndicator[int64(items[i].ID)]
		if ok {
			items[i].YearValues = yearValues
		}
	}

	totalPages := total / limit
	if total%limit != 0 {
		totalPages++
	}

	c.JSON(200, planningPeriodListResponse{
		Items: items,
		Meta: listMeta{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	})
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

	tx, err := h.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to start transaction"})
		return
	}
	defer tx.Rollback()

	var id int64
	err = tx.QueryRow(`
		INSERT INTO planning_period_indicators (target_indicator, unit, year_values, created_at, updated_at)
		VALUES ($1, $2, $3::jsonb, NOW(), NOW())
		RETURNING id
	`, req.TargetIndicator, req.Unit, yearValuesJSON).Scan(&id)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to create indicator"})
		return
	}

	if err := replaceIndicatorYearValuesTx(tx, id, req.YearValues); err != nil {
		c.JSON(500, errorResponse{Error: "failed to save yearly targets"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to commit indicator"})
		return
	}

	item, err := h.fetchIndicatorByID(int(id))
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to load created indicator"})
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
		next := strings.TrimSpace(*req.TargetIndicator)
		if next == "" {
			c.JSON(400, errorResponse{Error: "target_indicator cannot be empty"})
			return
		}
		item.TargetIndicator = next
	}

	if req.Unit != nil {
		next := strings.TrimSpace(*req.Unit)
		if next == "" {
			c.JSON(400, errorResponse{Error: "unit cannot be empty"})
			return
		}
		item.Unit = next
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

	tx, err := h.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to start transaction"})
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		UPDATE planning_period_indicators
		SET target_indicator = $1,
		    unit = $2,
		    year_values = $3::jsonb,
		    updated_at = NOW()
		WHERE id = $4
	`, item.TargetIndicator, item.Unit, yearValuesJSON, id)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to update indicator"})
		return
	}

	if req.YearValues != nil {
		if err := replaceIndicatorYearValuesTx(tx, int64(id), item.YearValues); err != nil {
			c.JSON(500, errorResponse{Error: "failed to update yearly targets"})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to commit update"})
		return
	}

	updated, err := h.fetchIndicatorByID(id)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to load updated indicator"})
		return
	}

	c.JSON(200, updated)
}

type planningPeriodImportRow struct {
	TargetIndicator string
	Unit            string
	YearValues      map[string]string
}

func (h *Handler) importExcel(c *gin.Context) {
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
		c.JSON(500, errorResponse{Error: "failed to prepare planning period storage"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, errorResponse{Error: "excel file is required"})
		return
	}

	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(fileHeader.Filename)))
	if ext != ".xlsx" {
		c.JSON(400, errorResponse{Error: "only .xlsx file is supported"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(400, errorResponse{Error: "failed to open uploaded file"})
		return
	}
	defer file.Close()

	workbook, err := excelize.OpenReader(file)
	if err != nil {
		c.JSON(400, errorResponse{Error: "failed to parse excel file"})
		return
	}
	defer workbook.Close()

	rowsToImport, skipped, err := parsePlanningPeriodWorkbook(workbook)
	if err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}
	if len(rowsToImport) == 0 {
		c.JSON(400, errorResponse{Error: "file has no valid data rows"})
		return
	}

	tx, err := h.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to start transaction"})
		return
	}
	defer tx.Rollback()

	created := 0
	updated := 0

	for _, row := range rowsToImport {
		yearValuesJSON, marshalErr := json.Marshal(row.YearValues)
		if marshalErr != nil {
			c.JSON(500, errorResponse{Error: "failed to serialize year values"})
			return
		}

		var id int64
		scanErr := tx.QueryRow(`
			SELECT id
			FROM planning_period_indicators
			WHERE target_indicator = $1
			  AND unit = $2
			LIMIT 1
		`, row.TargetIndicator, row.Unit).Scan(&id)
		if scanErr != nil && !errors.Is(scanErr, sql.ErrNoRows) {
			c.JSON(500, errorResponse{Error: "failed to find existing indicator"})
			return
		}

		if errors.Is(scanErr, sql.ErrNoRows) {
			if err := tx.QueryRow(`
				INSERT INTO planning_period_indicators (target_indicator, unit, year_values, created_at, updated_at)
				VALUES ($1, $2, $3::jsonb, NOW(), NOW())
				RETURNING id
			`, row.TargetIndicator, row.Unit, yearValuesJSON).Scan(&id); err != nil {
				c.JSON(500, errorResponse{Error: "failed to create imported indicator"})
				return
			}
			created++
		} else {
			if _, err := tx.Exec(`
				UPDATE planning_period_indicators
				SET year_values = $1::jsonb,
				    updated_at = NOW()
				WHERE id = $2
			`, yearValuesJSON, id); err != nil {
				c.JSON(500, errorResponse{Error: "failed to update imported indicator"})
				return
			}
			updated++
		}

		if err := replaceIndicatorYearValuesTx(tx, id, row.YearValues); err != nil {
			c.JSON(500, errorResponse{Error: "failed to save imported yearly targets"})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to commit import"})
		return
	}

	c.JSON(200, importPlanningPeriodResponse{
		Created: created,
		Updated: updated,
		Skipped: skipped,
	})
}

func parsePlanningPeriodWorkbook(workbook *excelize.File) ([]planningPeriodImportRow, int, error) {
	sheetName := workbook.GetSheetName(0)
	if strings.TrimSpace(sheetName) == "" {
		return nil, 0, fmt.Errorf("excel file has no sheets")
	}

	rawRows, err := workbook.GetRows(sheetName)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read sheet rows")
	}
	if len(rawRows) == 0 {
		return nil, 0, fmt.Errorf("excel sheet is empty")
	}

	headerRowIndex := -1
	targetIndex := -1
	unitIndex := -1
	yearColumns := map[int]int{}

	for rowIdx, row := range rawRows {
		tmpTarget := -1
		tmpUnit := -1
		tmpYearColumns := map[int]int{}

		for cellIdx, cell := range row {
			normalized := normalizeHeaderCell(cell)
			switch {
			case strings.Contains(normalized, "целевой индикатор"):
				tmpTarget = cellIdx
			case strings.Contains(normalized, "ед изм"):
				tmpUnit = cellIdx
			}

			year, yearErr := strconv.Atoi(strings.TrimSpace(cell))
			if yearErr == nil && year >= 2000 && year <= 2100 {
				tmpYearColumns[cellIdx] = year
			}
		}

		if tmpTarget >= 0 && tmpUnit >= 0 && len(tmpYearColumns) > 0 {
			headerRowIndex = rowIdx
			targetIndex = tmpTarget
			unitIndex = tmpUnit
			yearColumns = tmpYearColumns
			break
		}
	}

	if headerRowIndex < 0 {
		return nil, 0, fmt.Errorf("header row not found (required: Целевой индикатор, ед. изм., years)")
	}

	rows := make([]planningPeriodImportRow, 0)
	skipped := 0

	for rowIdx := headerRowIndex + 1; rowIdx < len(rawRows); rowIdx++ {
		row := rawRows[rowIdx]
		target := strings.TrimSpace(cellAt(row, targetIndex))
		unit := strings.TrimSpace(cellAt(row, unitIndex))

		yearValues := map[string]string{}
		for colIdx, year := range yearColumns {
			rawValue := strings.TrimSpace(cellAt(row, colIdx))
			if rawValue == "" {
				continue
			}

			parsedValue := normalizeExcelCellValue(rawValue)
			if parsedValue == "" {
				continue
			}
			yearValues[strconv.Itoa(year)] = parsedValue
		}

		if target == "" && unit == "" && len(yearValues) == 0 {
			skipped++
			continue
		}

		if target == "" || unit == "" {
			return nil, 0, fmt.Errorf("target indicator and unit are required at row %d", rowIdx+1)
		}
		if len(yearValues) == 0 {
			skipped++
			continue
		}

		if err := validateYearValues(yearValues); err != nil {
			return nil, 0, fmt.Errorf("invalid year values at row %d: %s", rowIdx+1, err.Error())
		}

		rows = append(rows, planningPeriodImportRow{
			TargetIndicator: target,
			Unit:            unit,
			YearValues:      yearValues,
		})
	}

	return rows, skipped, nil
}

func normalizeHeaderCell(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, ".", "")
	normalized = strings.ReplaceAll(normalized, "ё", "е")
	normalized = strings.Join(strings.Fields(normalized), " ")
	return normalized
}

func cellAt(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return row[index]
}

func normalizeExcelCellValue(raw string) string {
	normalized := strings.TrimSpace(raw)
	normalized = strings.ReplaceAll(normalized, "\u00a0", " ")
	normalized = strings.Join(strings.Fields(normalized), " ")
	return normalized
}

func (h *Handler) fetchIndicatorByID(id int) (model.PlanningPeriodIndicator, error) {
	var item model.PlanningPeriodIndicator
	err := h.db.QueryRow(`
		SELECT id, target_indicator, unit, created_at, updated_at
		FROM planning_period_indicators
		WHERE id = $1
		LIMIT 1
	`, id).Scan(
		&item.ID,
		&item.TargetIndicator,
		&item.Unit,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return model.PlanningPeriodIndicator{}, err
	}

	valuesByIndicator, err := h.loadYearValuesByIndicatorIDs([]int64{int64(item.ID)})
	if err != nil {
		return model.PlanningPeriodIndicator{}, err
	}
	item.YearValues = valuesByIndicator[int64(item.ID)]
	if item.YearValues == nil {
		item.YearValues = map[string]string{}
	}

	return item, nil
}

func (h *Handler) loadYearValuesByIndicatorIDs(ids []int64) (map[int64]map[string]string, error) {
	result := make(map[int64]map[string]string)
	if len(ids) == 0 {
		return result, nil
	}

	placeholders := make([]string, 0, len(ids))
	args := make([]any, 0, len(ids))
	for i, id := range ids {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
		args = append(args, id)
	}

	rows, err := h.db.Query(`
		SELECT indicator_id, year, planned_value
		FROM indicator_year_targets
		WHERE indicator_id IN (`+strings.Join(placeholders, ",")+`)
		ORDER BY indicator_id ASC, year ASC
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var indicatorID int64
		var year int
		var plannedValue string
		if err := rows.Scan(&indicatorID, &year, &plannedValue); err != nil {
			return nil, err
		}

		values, ok := result[indicatorID]
		if !ok {
			values = map[string]string{}
			result[indicatorID] = values
		}
		values[strconv.Itoa(year)] = plannedValue
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func replaceIndicatorYearValuesTx(tx *sql.Tx, indicatorID int64, values map[string]string) error {
	if _, err := tx.Exec(`
		DELETE FROM indicator_year_targets
		WHERE indicator_id = $1
	`, indicatorID); err != nil {
		return err
	}

	type yearValue struct {
		year  int
		value string
	}
	normalized := make([]yearValue, 0, len(values))
	for key, value := range values {
		year, err := strconv.Atoi(strings.TrimSpace(key))
		if err != nil {
			return fmt.Errorf("invalid year key: %s", key)
		}
		normalizedValue := strings.TrimSpace(value)
		if normalizedValue == "" {
			return fmt.Errorf("empty value for year: %d", year)
		}
		normalized = append(normalized, yearValue{year: year, value: normalizedValue})
	}
	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].year < normalized[j].year
	})

	for _, entry := range normalized {
		if _, err := tx.Exec(`
			INSERT INTO indicator_year_targets (
				indicator_id,
				year,
				planned_value,
				created_at,
				updated_at
			)
			VALUES ($1, $2, $3, NOW(), NOW())
		`, indicatorID, entry.year, entry.value); err != nil {
			return err
		}
	}

	return nil
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

func validateYearValues(values map[string]string) error {
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

		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("empty value for year %s", key)
		}
	}

	return nil
}

package plan

import (
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	db *sql.DB
}

type Plan struct {
	ID        int       `json:"id"`
	Year      int       `json:"year"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type createPlanRequest struct {
	Year   int    `json:"year" binding:"required"`
	Status string `json:"status"`
}

type updatePlanStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type planListResponse struct {
	Items []Plan `json:"items"`
}

func RegisterRoutes(router gin.IRoutes, db *sql.DB) {
	h := &Handler{db: db}

	router.GET("/plans", h.listPlans)
	router.POST("/plans", h.createPlan)
	router.GET("/plans/years", h.listPlanYears)
	router.GET("/plans/indicators", h.listPlanIndicators)
	router.PUT("/plans/indicators/:indicator_id", h.upsertPlanIndicator)
	router.POST("/plans/indicators/:indicator_id/report", h.submitPlanIndicatorReport)
	router.GET("/plans/reports", h.listPlanReports)
	router.GET("/plan-records/:id", h.getPlan)
	router.PATCH("/plan-records/:id/status", h.updatePlanStatus)
}

// listPlans godoc
// @Summary List plans
// @Description Returns yearly operational plans
// @Tags plans
// @Produce json
// @Security BearerAuth
// @Success 200 {object} planListResponse
// @Router /plans [get]
func (h *Handler) listPlans(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT id, year, status, created_at
		FROM plans
		ORDER BY year DESC, id DESC
	`)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to load plans"})
		return
	}
	defer rows.Close()

	items := make([]Plan, 0)
	for rows.Next() {
		var plan Plan
		if err := rows.Scan(&plan.ID, &plan.Year, &plan.Status, &plan.CreatedAt); err != nil {
			c.JSON(500, errorResponse{Error: "failed to parse plans"})
			return
		}
		items = append(items, plan)
	}

	if err := rows.Err(); err != nil {
		c.JSON(500, errorResponse{Error: "failed to iterate plans"})
		return
	}

	c.JSON(200, planListResponse{Items: items})
}

// createPlan godoc
// @Summary Create plan
// @Description Creates a plan for selected year
// @Tags plans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param payload body createPlanRequest true "Plan payload"
// @Success 201 {object} Plan
// @Failure 400 {object} errorResponse
// @Router /plans [post]
func (h *Handler) createPlan(c *gin.Context) {
	var req createPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	status := req.Status
	if status == "" {
		status = "draft"
	}

	var plan Plan
	err := h.db.QueryRow(`
		INSERT INTO plans (year, status, created_at)
		VALUES ($1, $2, NOW())
		RETURNING id, year, status, created_at
	`, req.Year, status).Scan(&plan.ID, &plan.Year, &plan.Status, &plan.CreatedAt)
	if err != nil {
		c.JSON(500, errorResponse{Error: "failed to create plan"})
		return
	}

	c.JSON(201, plan)
}

// getPlan godoc
// @Summary Get plan
// @Description Returns plan by ID
// @Tags plans
// @Produce json
// @Security BearerAuth
// @Param id path int true "Plan ID"
// @Success 200 {object} Plan
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /plan-records/{id} [get]
func (h *Handler) getPlan(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, errorResponse{Error: "invalid plan id"})
		return
	}

	plan, err := h.fetchPlanByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(404, errorResponse{Error: "plan not found"})
			return
		}

		c.JSON(500, errorResponse{Error: "failed to load plan"})
		return
	}

	c.JSON(200, plan)
}

// updatePlanStatus godoc
// @Summary Update plan status
// @Description Changes status of selected plan
// @Tags plans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Plan ID"
// @Param payload body updatePlanStatusRequest true "Status payload"
// @Success 200 {object} Plan
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /plan-records/{id}/status [patch]
func (h *Handler) updatePlanStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, errorResponse{Error: "invalid plan id"})
		return
	}

	var req updatePlanStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, errorResponse{Error: err.Error()})
		return
	}

	var plan Plan
	err = h.db.QueryRow(`
		UPDATE plans
		SET status = $1
		WHERE id = $2
		RETURNING id, year, status, created_at
	`, req.Status, id).Scan(&plan.ID, &plan.Year, &plan.Status, &plan.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(404, errorResponse{Error: "plan not found"})
			return
		}

		c.JSON(500, errorResponse{Error: "failed to update plan status"})
		return
	}

	c.JSON(200, plan)
}

func (h *Handler) fetchPlanByID(id int) (Plan, error) {
	var plan Plan
	err := h.db.QueryRow(`
		SELECT id, year, status, created_at
		FROM plans
		WHERE id = $1
	`, id).Scan(&plan.ID, &plan.Year, &plan.Status, &plan.CreatedAt)
	if err != nil {
		return Plan{}, err
	}

	return plan, nil
}

package handler

import (
	"database/sql"

	"OperationPlan/internal/auth"
	"OperationPlan/internal/middleware"
	"OperationPlan/internal/period"
	"OperationPlan/internal/plan"
	"OperationPlan/internal/user"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func RegisterRoutes(router *gin.Engine, db *sql.DB) {
	router.GET("/healthz", healthz)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	auth.RegisterRoutes(router, db)

	protected := router.Group("/")
	protected.Use(middleware.AuthRequired(db))

	plan.RegisterRoutes(protected, db)
	period.RegisterRoutes(protected, db)

	adminProtected := protected.Group("/")
	adminProtected.Use(middleware.RequireRoles("admin"))
	user.RegisterRoutes(adminProtected, db)
}

type healthResponse struct {
	Status string `json:"status"`
}

// healthz godoc
// @Summary Health check
// @Description Returns server health state
// @Tags system
// @Produce json
// @Success 200 {object} healthResponse
// @Router /healthz [get]
func healthz(c *gin.Context) {
	c.JSON(200, healthResponse{Status: "ok"})
}

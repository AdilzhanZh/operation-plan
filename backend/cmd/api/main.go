package main

import (
	_ "OperationPlan/docs"
	"OperationPlan/internal/config"
	"OperationPlan/internal/database"
	"OperationPlan/internal/handler"
	"OperationPlan/internal/pkg/logger"
	"OperationPlan/internal/server"
	"log"
	"log/slog"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// @title Oper Plan API
// @version 1.0
// @description API for planning, execution control and reporting in Oper Plan.
// @BasePath /
// @schemes http https
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %s", err.Error())
	}

	slogger := logger.New(cfg.LogLevel)
	slog.SetDefault(slogger)

	db, err := database.OpenPostgres(cfg)
	if err != nil {
		log.Fatalf("failed to connect database: %s", err.Error())
	}

	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("failed to run migrations: %s", err.Error())
	}

	slog.Info("database connected and migrated", "host", cfg.DBHost, "port", cfg.DBPort, "dbname", cfg.DBName)

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to create sql db handle: %s", err.Error())
	}

	if err := database.RunSQLMigrations(sqlDB); err != nil {
		log.Fatalf("failed to run SQL migrations: %s", err.Error())
	}

	r := gin.New()
	r.Use(
		cors.New(cors.Config{
			AllowOrigins: cfg.CORSAllowedOrigins,
			AllowMethods: []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
			MaxAge:       12 * time.Hour,
		}),
		gin.Logger(),
		gin.Recovery(),
	)

	handler.RegisterRoutes(r, sqlDB, cfg)

	srv := server.New(r, cfg.Port)
	if err := srv.Run(); err != nil {
		log.Fatalf("server failed: %s", err.Error())
	}
}

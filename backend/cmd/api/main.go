package main

import (
	"OperationPlan/internal/config"
	"OperationPlan/internal/pkg/logger"
	"OperationPlan/internal/server"
	"log"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %s", err.Error())
	}

	slogger := logger.New(cfg.LogLevel)
	slog.SetDefault(slogger)

	r := gin.New()

	r.GET("/test", func(context *gin.Context) {
		context.JSON(http.StatusOK, gin.H{"message": "It's all right"})
	})

	srv := server.New(r, cfg.Port)
	srv.Run()
}

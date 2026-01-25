package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"dist_task/internal/api/handler"
	"dist_task/internal/config"
	"dist_task/internal/engine"
	"dist_task/internal/repository"
	"dist_task/pkg/logger"

	"github.com/gin-gonic/gin"
)

func main() {
	cfgPath := "configs/app.toml"
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		cfgPath = envPath
	}

	cfg := config.MustLoad(cfgPath)

	logger.Init(&cfg.Log)

	if err := repository.Init(cfg.Database.DSN()); err != nil {
		log.Fatalf("init database failed: %v", err)
	}

	flowRepo := &repository.FlowRepository{}
	instanceRepo := &repository.InstanceRepository{}
	taskRepo := &repository.TaskRepository{}
	exceptionRepo := &repository.ExceptionRepository{}
	logRepo := &repository.LogRepository{}

	eng := engine.NewEngine(instanceRepo, taskRepo, exceptionRepo, logRepo)

	h := handler.NewHandler(flowRepo, instanceRepo, taskRepo, exceptionRepo, logRepo, eng)

	r := gin.Default()

	r.GET("/health", h.HealthCheck)

	v1 := r.Group("/api/v1")
	{
		flows := v1.Group("/flows")
		{
			flows.POST("", h.CreateFlow)
			flows.GET("", h.ListFlows)
			flows.GET("/:id", h.GetFlow)
		}

		transactions := v1.Group("/transactions")
		{
			transactions.POST("", h.StartTransaction)
			transactions.GET("/:id", h.GetTransaction)
		}

		exceptions := v1.Group("/exceptions")
		{
			exceptions.GET("", h.ListExceptions)
		}
	}

	addr := fmt.Sprintf("%s:%d", cfg.App.Host, cfg.App.Port)
	log.Printf("server starting on %s", addr)

	go func() {
		if err := r.Run(addr); err != nil {
			log.Fatalf("server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("server shutdown")
}

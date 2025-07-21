package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"test_wallet/internal/config"
	"test_wallet/internal/handlers"
	"test_wallet/internal/logging"
	"test_wallet/internal/repository"
	"test_wallet/internal/service"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("failed to load config:", err)
	}

	logger := logging.SetupLogger()

	gin.SetMode(gin.ReleaseMode)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	poolConfig, err := pgxpool.ParseConfig(cfg.DBURL)
	if err != nil {
		logger.Error("failed to parse db config", "err", err)
		os.Exit(1)
	}
	poolConfig.MaxConns = int32(cfg.DBMaxConns)
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		logger.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	repo := repository.NewWalletPGRepository(pool, logger)
	svc := service.NewWalletService(repo, logger)
	hanlder := handlers.NewWalletHTTPHandler(svc)

	r := gin.Default()
	hanlder.RegisterRoutes(r)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		logger.Info("Starting server", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil {
			logger.Error("Server failed", "err", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server")

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(ctxShutdown); err != nil {
		logger.Error("Server forced to shutdown", "err", err)
	}
	logger.Info("Server exiting")
}

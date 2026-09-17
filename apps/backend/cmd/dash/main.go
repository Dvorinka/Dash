// Command dash is the single-binary entrypoint: API + embedded UI.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/tdvorak/dash/internal/api"
	"github.com/tdvorak/dash/internal/db"
	"github.com/tdvorak/dash/internal/web"
)

func main() {
	dev := os.Getenv("DASH_DEV") == "1"

	var logger *zap.Logger
	if dev {
		logger = zap.Must(zap.NewDevelopment())
	} else {
		gin.SetMode(gin.ReleaseMode)
		logger = zap.Must(zap.NewProduction())
	}
	defer func() { _ = logger.Sync() }()

	dataDir := envOr("DASH_DATA_DIR", "./data")
	if err := os.MkdirAll(filepath.Join(dataDir, "icons"), 0o755); err != nil {
		logger.Fatal("create data dir", zap.Error(err))
	}

	sqlDB, err := db.Open(filepath.Join(dataDir, "dash.db"))
	if err != nil {
		logger.Fatal("open database", zap.Error(err))
	}
	defer func() { _ = sqlDB.Close() }()

	iconsDir := filepath.Join(dataDir, "icons")
	router := api.NewRouter(logger, sqlDB, iconsDir)
	if err := web.Register(router); err != nil {
		logger.Fatal("register embedded UI", zap.Error(err))
	}

	addr := ":" + envOr("DASH_PORT", "8080")
	srv := &http.Server{Addr: addr, Handler: router, ReadHeaderTimeout: 10 * time.Second}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("listening", zap.String("addr", addr), zap.String("data", dataDir))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("serve", zap.Error(err))
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown", zap.Error(err))
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

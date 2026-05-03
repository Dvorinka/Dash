package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"

	"dash/backend/internal/config"
	"dash/backend/internal/httpapi"
	"dash/backend/internal/store"
)

func main() {
	log, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer func() { _ = log.Sync() }()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("load config", zap.Error(err))
	}
	if cfg.AppEnv == "development" {
		devLog, err := zap.NewDevelopment()
		if err == nil {
			log = devLog
		}
	}

	if err := os.MkdirAll(cfg.IconDir(), 0o755); err != nil {
		log.Fatal("create data dirs", zap.Error(err))
	}
	if err := runMigrations(cfg); err != nil {
		log.Fatal("run migrations", zap.Error(err))
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal("connect database", zap.Error(err))
	}
	defer pool.Close()

	st := store.New(pool)
	router := httpapi.NewRouter(cfg, log, st)
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("backend listening", zap.String("addr", cfg.HTTPAddr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("server failed", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown failed", zap.Error(err))
	}
}

func runMigrations(cfg config.Config) error {
	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Up(db, cfg.MigrationsDir)
}

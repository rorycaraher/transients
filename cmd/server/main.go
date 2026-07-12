package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rorycaraher/transients/internal/config"
	"github.com/rorycaraher/transients/internal/db"
	"github.com/rorycaraher/transients/internal/ingest"
	"github.com/rorycaraher/transients/internal/r2"
	"github.com/rorycaraher/transients/internal/store"
	"github.com/rorycaraher/transients/internal/web"
)

const httpShutdownTimeout = 10 * time.Second

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		log.Error("config load failed", "err", err)
		os.Exit(1)
	}

	conn, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Error("db open failed", "err", err)
		os.Exit(1)
	}
	defer conn.Close()

	st := store.New(conn)
	r2c := r2.New(cfg.R2AccountID, cfg.R2AccessKeyID, cfg.R2SecretAccessKey, cfg.R2Bucket)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	poller := ingest.NewPoller(cfg.R2AccountID, cfg.CFAPIToken, cfg.CFQueueID, st, r2c, cfg.PollInterval, log)
	go poller.Run(ctx)

	srv, err := web.NewServer(cfg, st, r2c, log)
	if err != nil {
		log.Error("web server init failed", "err", err)
		os.Exit(1)
	}

	httpServer := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: srv.Mux(),
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), httpShutdownTimeout)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	log.Info("listening", "addr", httpServer.Addr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("server failed", "err", err)
		os.Exit(1)
	}
}

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"pipefy-integration/internal/env"
	"pipefy-integration/internal/repository"
	"pipefy-integration/internal/repository/postgres"
	"pipefy-integration/internal/service"
	"pipefy-integration/internal/validate"
	"syscall"
	"time"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("starting application")
	
	if err := run(); err != nil {
		slog.Error("application setup fail", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	dsn := env.Get("DSN", "postgres://username:password@postgres:5432/clients")
	db, err := connectToDB(ctx, dsn, 3, 2*time.Second)
	if err != nil {
		return err
	}
	defer db.Close()
	
	slog.Info("database connection successfully!")

	databaseSvc := service.NewDatabaseService(db)

	app := application{
		port:      env.Get("HTTP_PORT", "8080"),
		validator: validate.New(),
		databaseService: databaseSvc,
	}

	mux := app.mount()

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", app.port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("server start listening", slog.String("port", app.port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server fail to listen", slog.String("error", err.Error()))
			cancel()
		}
	}()

	<-ctx.Done()

	slog.Info("server shutting down...")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()

	srv.Shutdown(shutCtx)
	return nil
}

func connectToDB(
	ctx context.Context, dsn string, attempts int, delay time.Duration,
) (repository.DatabaseStore, error) {
	var err error
	for i := range attempts {
		conn, e := postgres.NewStore(ctx, dsn)
		if e == nil {
			return conn, nil
		}

		err = e
		slog.Warn("database not ready", slog.Int("attempt", i+1), slog.String("error", e.Error()))
		time.Sleep(delay)
	}

	return nil, err
}

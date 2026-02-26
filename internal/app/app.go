package app

import (
	"context"
	"fmt"
	"time"

	"goadmin/internal/config"
	"goadmin/internal/core"
	"goadmin/internal/modules/host"
	"goadmin/internal/storage"
	"goadmin/internal/storage/sqlite"
)

// App агрегирует зависимости ядра.
type App struct {
	Registry   *core.Registry
	Transports *core.TransportManager
	Authorizer core.Authorizer
	Store      storage.Store
	Config     config.Config
}

// NewApp строит приложение: реестр модулей и хранилище.
func NewApp(ctx context.Context, cfg config.Config) (*App, error) {
	r := core.NewRegistry()
	if err := r.Register(ctx, &host.Module{}); err != nil {
		return nil, fmt.Errorf("register host module: %w", err)
	}

	st, err := sqlite.Open(cfg.SQLite.Path)
	if err != nil {
		return nil, fmt.Errorf("open storage: %w", err)
	}

	return &App{
		Registry:   r,
		Transports: core.NewTransportManager(),
		Authorizer: core.NewAllowlistAuthorizer(cfg.Security.AuthAllowlist),
		Store:      st,
		Config:     cfg,
	}, nil
}

// Close высвобождает ресурсы приложения.
func (a *App) Close() error {
	if a.Store != nil {
		return a.Store.Close()
	}
	return nil
}

// Serve запускает планировщик периодического сбора метрик.
func (a *App) Serve(ctx context.Context) error {
	interval := time.Duration(a.Config.Scheduler.IntervalSeconds) * time.Second
	if interval <= 0 {
		interval = time.Minute
	}
	sched := core.NewScheduler(interval)

	sched.Add(func(jobCtx context.Context) error {
		runCtx, cancel := context.WithTimeout(jobCtx, 3*time.Second)
		defer cancel()

		resp, err := a.Registry.Execute(runCtx, "host", "status", nil)
		if err != nil {
			return fmt.Errorf("host status: %w", err)
		}
		payload, err := sqlite.MarshalPayload(resp.Data)
		if err != nil {
			return err
		}
		return a.Store.SaveMetric(jobCtx, storage.MetricRecord{Module: "host", Payload: payload})
	})

	sched.Start(ctx)
	return ctx.Err()
}

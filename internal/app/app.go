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
	"goadmin/internal/transports/common"
	"goadmin/internal/transports/maxbot"
	"goadmin/internal/transports/telegram"
	"goadmin/internal/transports/web"
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

	authz := core.NewAllowlistAuthorizer(cfg.Security.AuthAllowlist)
	transports := core.NewTransportManager()
	limiter := common.NewRateLimiter(5, time.Second)

	audit := st
	tg := telegram.NewAdapter(r, authz, limiter, audit)
	mx := maxbot.NewAdapter(r, authz, limiter, audit)
	if err := transports.Register(tg); err != nil {
		return nil, fmt.Errorf("register telegram transport: %w", err)
	}
	if err := transports.Register(mx); err != nil {
		return nil, fmt.Errorf("register maxbot transport: %w", err)
	}
	if cfg.Web.Enabled {
		tokens := make([]web.TokenEntry, 0, len(cfg.Web.Auth.Tokens))
		for _, token := range cfg.Web.Auth.Tokens {
			tokens = append(tokens, web.TokenEntry{
				ID:          token.ID,
				TokenSHA256: token.TokenSHA256,
				Subject:     token.Subject,
				Roles:       token.Roles,
				Enabled:     token.Enabled,
			})
		}

		webAdapter := web.NewAdapter(r, authz, st, web.Config{
			ListenAddr:               cfg.Web.ListenAddr,
			ReadTimeout:              time.Duration(cfg.Web.ReadTimeoutMS) * time.Millisecond,
			WriteTimeout:             time.Duration(cfg.Web.WriteTimeoutMS) * time.Millisecond,
			RequestTimeout:           time.Duration(cfg.Web.RequestTimeoutMS) * time.Millisecond,
			ShutdownTimeout:          time.Duration(cfg.Web.ShutdownTimeoutS) * time.Second,
			MaxRequestBody:           cfg.Web.MaxBodyBytes,
			AuthMode:                 cfg.Web.Auth.Mode,
			AllowLegacySubjectHeader: cfg.Web.Auth.AllowLegacySubjectHeader,
			Tokens:                   tokens,
			CORSAllowedOrigins:       cfg.Web.CORS.AllowedOrigins,
			CORSAllowedMethods:       cfg.Web.CORS.AllowedMethods,
			CORSAllowedHeaders:       cfg.Web.CORS.AllowedHeaders,
		})
		if err := transports.Register(webAdapter); err != nil {
			return nil, fmt.Errorf("register web transport: %w", err)
		}
	}

	return &App{
		Registry:   r,
		Transports: transports,
		Authorizer: authz,
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
	if err := a.Transports.StartAll(ctx); err != nil {
		return fmt.Errorf("start transports: %w", err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = a.Transports.StopAll(stopCtx)
	}()

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

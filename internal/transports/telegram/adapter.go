package telegram

import (
	"context"
	"sync"

	"goadmin/internal/core"
	"goadmin/internal/transports/common"
)

// Adapter предоставляет transport-слой для Telegram.
type Adapter struct {
	svc     *common.Service
	mu      sync.Mutex
	running bool
}

// NewAdapter создает Telegram адаптер.
func NewAdapter(registry *core.Registry, authorizer core.Authorizer, limiter *common.RateLimiter, audit common.AuditSink) *Adapter {
	return &Adapter{
		svc: &common.Service{
			Source:      "telegram",
			Registry:    registry,
			Authorizer:  authorizer,
			RateLimiter: limiter,
			AuditSink:   audit,
		},
	}
}

func (a *Adapter) Name() string { return "telegram" }

func (a *Adapter) Start(ctx context.Context) error {
	a.mu.Lock()
	a.running = true
	a.mu.Unlock()
	return nil
}

func (a *Adapter) Stop(ctx context.Context) error {
	a.mu.Lock()
	a.running = false
	a.mu.Unlock()
	return nil
}

// HandleCommand принимает команду в чат-формате и исполняет через core.
func (a *Adapter) HandleCommand(ctx context.Context, userID, text string) (core.Response, error) {
	return a.svc.ExecuteText(ctx, userID, text)
}

package core

import (
	"context"
	"errors"
	"fmt"
)

var (
	errProviderExists   = errors.New("provider already registered")
	errUnknownProvider  = errors.New("unknown provider")
	errInvalidArguments = errors.New("invalid arguments")
)

// Registry хранит зарегистрированные модули и выполняет команды.
type Registry struct {
	providers map[string]CommandProvider
}

// NewRegistry создает пустой реестр модулей.
func NewRegistry() *Registry {
	return &Registry{providers: make(map[string]CommandProvider)}
}

// Register добавляет модуль; имя должно быть уникальным.
func (r *Registry) Register(ctx context.Context, provider CommandProvider) error {
	if provider == nil {
		return fmt.Errorf("provider is nil: %w", errInvalidArguments)
	}
	name := provider.Name()
	if name == "" {
		return fmt.Errorf("provider name is empty: %w", errInvalidArguments)
	}
	if _, exists := r.providers[name]; exists {
		return fmt.Errorf("%s: %w", name, errProviderExists)
	}
	if err := provider.Init(ctx); err != nil {
		return fmt.Errorf("init %s: %w", name, err)
	}
	r.providers[name] = provider
	return nil
}

// Execute вызывает модуль по имени.
func (r *Registry) Execute(ctx context.Context, module, cmd string, args []string) (Response, error) {
	prov, ok := r.providers[module]
	if !ok {
		return Response{Status: "error", ErrorCode: "module_not_found"}, fmt.Errorf("%s: %w", module, errUnknownProvider)
	}
	return prov.Execute(ctx, cmd, args)
}

// Providers возвращает список зарегистрированных модулей.
func (r *Registry) Providers() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

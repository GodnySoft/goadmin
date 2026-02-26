package core

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var (
	errTransportExists  = errors.New("transport already registered")
	errUnknownTransport = errors.New("unknown transport")
)

// TransportAdapter определяет жизненный цикл входного транспорта.
type TransportAdapter interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// TransportManager управляет запуском и остановкой транспортов.
type TransportManager struct {
	mu         sync.Mutex
	transports map[string]TransportAdapter
}

// NewTransportManager создает пустой менеджер транспортов.
func NewTransportManager() *TransportManager {
	return &TransportManager{transports: make(map[string]TransportAdapter)}
}

// Register добавляет транспорт; имена должны быть уникальны.
func (m *TransportManager) Register(adapter TransportAdapter) error {
	if adapter == nil {
		return fmt.Errorf("transport is nil: %w", errInvalidArguments)
	}
	name := adapter.Name()
	if name == "" {
		return fmt.Errorf("transport name is empty: %w", errInvalidArguments)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.transports[name]; exists {
		return fmt.Errorf("%s: %w", name, errTransportExists)
	}
	m.transports[name] = adapter
	return nil
}

// StartAll запускает все зарегистрированные транспорты.
func (m *TransportManager) StartAll(ctx context.Context) error {
	m.mu.Lock()
	list := make([]TransportAdapter, 0, len(m.transports))
	for _, tr := range m.transports {
		list = append(list, tr)
	}
	m.mu.Unlock()

	for _, tr := range list {
		if err := tr.Start(ctx); err != nil {
			return fmt.Errorf("start transport %s: %w", tr.Name(), err)
		}
	}
	return nil
}

// StopAll останавливает все зарегистрированные транспорты.
func (m *TransportManager) StopAll(ctx context.Context) error {
	m.mu.Lock()
	list := make([]TransportAdapter, 0, len(m.transports))
	for _, tr := range m.transports {
		list = append(list, tr)
	}
	m.mu.Unlock()

	for _, tr := range list {
		if err := tr.Stop(ctx); err != nil {
			return fmt.Errorf("stop transport %s: %w", tr.Name(), err)
		}
	}
	return nil
}

// StopOne останавливает конкретный транспорт по имени.
func (m *TransportManager) StopOne(ctx context.Context, name string) error {
	m.mu.Lock()
	tr, ok := m.transports[name]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("%s: %w", name, errUnknownTransport)
	}
	if err := tr.Stop(ctx); err != nil {
		return fmt.Errorf("stop transport %s: %w", tr.Name(), err)
	}
	return nil
}

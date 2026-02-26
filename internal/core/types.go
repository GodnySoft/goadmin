package core

import "context"

// Response описывает унифицированный результат выполнения команды.
type Response struct {
	Status    string      `json:"status"`
	Data      interface{} `json:"data,omitempty"`
	ErrorCode string      `json:"error_code,omitempty"`
}

// CommandProvider определяет контракт для модулей.
type CommandProvider interface {
	Name() string
	Init(ctx context.Context) error
	Execute(ctx context.Context, cmd string, args []string) (Response, error)
}

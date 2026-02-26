package common

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"goadmin/internal/core"
)

var (
	errEmptyCommand = errors.New("empty command")
	errRateLimited  = errors.New("rate limit exceeded")
)

// Service объединяет общий пайплайн command->authz->ratelimit->core.
type Service struct {
	Source      string
	Registry    *core.Registry
	Authorizer  core.Authorizer
	RateLimiter *RateLimiter
}

// ExecuteText парсит команду транспорта и вызывает core-модуль.
func (s *Service) ExecuteText(ctx context.Context, subjectID, text string) (core.Response, error) {
	module, command, args, err := ParseTextCommand(text)
	if err != nil {
		return core.Response{Status: "error", ErrorCode: "bad_command"}, err
	}
	subject := core.Subject{Source: s.Source, ID: subjectID}
	action := core.Action{Module: module, Command: command}
	if err := s.Authorizer.Authorize(subject, action); err != nil {
		return core.Response{Status: "error", ErrorCode: "access_denied"}, err
	}
	if s.RateLimiter != nil {
		if !s.RateLimiter.Allow(fmt.Sprintf("%s:%s", s.Source, subjectID), time.Now()) {
			return core.Response{Status: "error", ErrorCode: "rate_limited"}, errRateLimited
		}
	}
	return s.Registry.Execute(ctx, module, command, args)
}

// ParseTextCommand переводит текст в (module, command, args).
// Формат: /module command arg1 arg2
func ParseTextCommand(text string) (string, string, []string, error) {
	t := strings.TrimSpace(text)
	if t == "" {
		return "", "", nil, errEmptyCommand
	}
	t = strings.TrimPrefix(t, "/")
	parts := strings.Fields(t)
	if len(parts) < 2 {
		return "", "", nil, fmt.Errorf("invalid command format: %w", errEmptyCommand)
	}
	module := parts[0]
	command := parts[1]
	args := []string{}
	if len(parts) > 2 {
		args = parts[2:]
	}
	return module, command, args, nil
}

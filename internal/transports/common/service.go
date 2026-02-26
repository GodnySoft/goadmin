package common

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"goadmin/internal/core"
	"goadmin/internal/storage"
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
	AuditSink   AuditSink
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
		s.writeAudit(ctx, subject, action, "denied", module, command, args)
		return core.Response{Status: "error", ErrorCode: "access_denied"}, err
	}
	if s.RateLimiter != nil {
		if !s.RateLimiter.Allow(fmt.Sprintf("%s:%s", s.Source, subjectID), time.Now()) {
			s.writeAudit(ctx, subject, action, "rate_limited", module, command, args)
			return core.Response{Status: "error", ErrorCode: "rate_limited"}, errRateLimited
		}
	}
	resp, execErr := s.Registry.Execute(ctx, module, command, args)
	status := "ok"
	if execErr != nil || resp.Status == "error" {
		status = "error"
	}
	s.writeAudit(ctx, subject, action, status, module, command, args)
	return resp, execErr
}

func (s *Service) writeAudit(ctx context.Context, subject core.Subject, action core.Action, status, module, command string, args []string) {
	if s.AuditSink == nil {
		return
	}
	_ = s.AuditSink.Write(ctx, storage.AuditEvent{
		Subject:   subject.ID,
		Action:    fmt.Sprintf("%s:%s", action.Module, action.Command),
		Source:    subject.Source,
		Status:    status,
		RequestID: newRequestID(),
		Payload:   buildAuditPayload(module, command, args),
	})
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

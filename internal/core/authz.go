package core

import "fmt"

// Subject описывает источник команды и его идентификатор.
type Subject struct {
	Source string
	ID     string
}

// Action описывает целевую операцию.
type Action struct {
	Module  string
	Command string
}

// Authorizer отвечает за решение доступа к действию.
type Authorizer interface {
	Authorize(subject Subject, action Action) error
}

// AllowlistAuthorizer реализует deny-by-default по source/id.
type AllowlistAuthorizer struct {
	allowed map[string]map[string]struct{}
}

// NewAllowlistAuthorizer создает authorizer из map[source][]id.
func NewAllowlistAuthorizer(src map[string][]string) *AllowlistAuthorizer {
	allowed := make(map[string]map[string]struct{}, len(src))
	for source, ids := range src {
		idSet := make(map[string]struct{}, len(ids))
		for _, id := range ids {
			if id == "" {
				continue
			}
			idSet[id] = struct{}{}
		}
		allowed[source] = idSet
	}
	return &AllowlistAuthorizer{allowed: allowed}
}

// Authorize возвращает ошибку, если subject не в allowlist.
func (a *AllowlistAuthorizer) Authorize(subject Subject, action Action) error {
	if subject.Source == "" || subject.ID == "" {
		return fmt.Errorf("empty subject: %w", errInvalidArguments)
	}
	bySource, ok := a.allowed[subject.Source]
	if !ok {
		return fmt.Errorf("source %s is not allowed", subject.Source)
	}
	if _, ok := bySource[subject.ID]; !ok {
		return fmt.Errorf("subject %s/%s is not allowed", subject.Source, subject.ID)
	}
	_ = action
	return nil
}

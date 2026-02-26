package core

import (
	"context"
	"errors"
	"testing"
)

type fakeProvider struct {
	name    string
	execErr error
}

func (f *fakeProvider) Name() string                   { return f.name }
func (f *fakeProvider) Init(ctx context.Context) error { return nil }
func (f *fakeProvider) Execute(ctx context.Context, cmd string, args []string) (Response, error) {
	if f.execErr != nil {
		return Response{Status: "error"}, f.execErr
	}
	return Response{Status: "ok", Data: cmd}, nil
}

func TestRegisterAndExecute(t *testing.T) {
	r := NewRegistry()
	ctx := context.Background()
	prov := &fakeProvider{name: "test"}
	if err := r.Register(ctx, prov); err != nil {
		t.Fatalf("register: %v", err)
	}
	resp, err := r.Execute(ctx, "test", "ping", nil)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if resp.Status != "ok" || resp.Data != "ping" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestDuplicateProvider(t *testing.T) {
	r := NewRegistry()
	ctx := context.Background()
	prov := &fakeProvider{name: "dup"}
	if err := r.Register(ctx, prov); err != nil {
		t.Fatalf("first register: %v", err)
	}
	if err := r.Register(ctx, prov); err == nil {
		t.Fatalf("expected error on duplicate register")
	}
}

func TestUnknownProvider(t *testing.T) {
	r := NewRegistry()
	ctx := context.Background()
	_, err := r.Execute(ctx, "none", "ping", nil)
	if err == nil {
		t.Fatalf("expected error for unknown provider")
	}
	if !errors.Is(err, errUnknownProvider) {
		t.Fatalf("expected errUnknownProvider, got %v", err)
	}
}

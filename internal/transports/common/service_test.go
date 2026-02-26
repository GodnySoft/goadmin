package common

import (
	"context"
	"testing"

	"goadmin/internal/core"
	"goadmin/internal/storage"
)

type fakeAuditSink struct {
	count int
}

func (f *fakeAuditSink) Write(ctx context.Context, ev storage.AuditEvent) error {
	f.count++
	return nil
}

func TestParseTextCommand(t *testing.T) {
	module, command, args, err := ParseTextCommand("/host status now")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if module != "host" || command != "status" {
		t.Fatalf("unexpected parsed command: %s %s", module, command)
	}
	if len(args) != 1 || args[0] != "now" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestParseTextCommandInvalid(t *testing.T) {
	_, _, _, err := ParseTextCommand("/host")
	if err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestServiceWritesAudit(t *testing.T) {
	sink := &fakeAuditSink{}
	svc := &Service{
		Source: "telegram",
		Registry: func() *core.Registry {
			r := core.NewRegistry()
			_ = r.Register(context.Background(), &testProvider{})
			return r
		}(),
		Authorizer:  core.NewAllowlistAuthorizer(map[string][]string{"telegram": {"1"}}),
		RateLimiter: NewRateLimiter(10, 0),
		AuditSink:   sink,
	}
	_, _ = svc.ExecuteText(context.Background(), "1", "/host status")
	if sink.count == 0 {
		t.Fatalf("expected audit sink to be called")
	}
}

type testProvider struct{}

func (t *testProvider) Name() string                   { return "host" }
func (t *testProvider) Init(ctx context.Context) error { return nil }
func (t *testProvider) Execute(ctx context.Context, cmd string, args []string) (core.Response, error) {
	return core.Response{Status: "ok"}, nil
}

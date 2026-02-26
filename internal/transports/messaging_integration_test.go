package transports

import (
	"context"
	"testing"
	"time"

	"goadmin/internal/core"
	"goadmin/internal/transports/common"
	"goadmin/internal/transports/telegram"
)

type testModule struct{}

func (m *testModule) Name() string                   { return "host" }
func (m *testModule) Init(ctx context.Context) error { return nil }
func (m *testModule) Execute(ctx context.Context, cmd string, args []string) (core.Response, error) {
	if cmd != "status" {
		return core.Response{Status: "error", ErrorCode: "unknown_command"}, nil
	}
	return core.Response{Status: "ok", Data: map[string]string{"result": "ok"}}, nil
}

func TestTelegramTransportPipeline(t *testing.T) {
	ctx := context.Background()
	r := core.NewRegistry()
	if err := r.Register(ctx, &testModule{}); err != nil {
		t.Fatalf("register module: %v", err)
	}
	authz := core.NewAllowlistAuthorizer(map[string][]string{"telegram": {"1001"}})
	limiter := common.NewRateLimiter(1, time.Second)
	tr := telegram.NewAdapter(r, authz, limiter)

	resp, err := tr.HandleCommand(ctx, "1001", "/host status")
	if err != nil {
		t.Fatalf("allowed command should pass: %v", err)
	}
	if resp.Status != "ok" {
		t.Fatalf("unexpected response status: %s", resp.Status)
	}

	_, err = tr.HandleCommand(ctx, "9999", "/host status")
	if err == nil {
		t.Fatalf("non-allowlisted subject must fail")
	}

	_, err = tr.HandleCommand(ctx, "1001", "/host status")
	if err == nil {
		t.Fatalf("rate-limit must block second immediate command")
	}
}

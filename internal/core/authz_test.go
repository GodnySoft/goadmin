package core

import "testing"

func TestAllowlistAuthorizerAuthorize(t *testing.T) {
	a := NewAllowlistAuthorizer(map[string][]string{
		"telegram": {"1001", "1002"},
	})
	if err := a.Authorize(Subject{Source: "telegram", ID: "1001"}, Action{Module: "host", Command: "status"}); err != nil {
		t.Fatalf("expected allow, got error: %v", err)
	}
}

func TestAllowlistAuthorizerDenyUnknownID(t *testing.T) {
	a := NewAllowlistAuthorizer(map[string][]string{
		"telegram": {"1001"},
	})
	if err := a.Authorize(Subject{Source: "telegram", ID: "9999"}, Action{Module: "host", Command: "status"}); err == nil {
		t.Fatalf("expected deny")
	}
}

func TestAllowlistAuthorizerDenyUnknownSource(t *testing.T) {
	a := NewAllowlistAuthorizer(map[string][]string{
		"telegram": {"1001"},
	})
	if err := a.Authorize(Subject{Source: "maxbot", ID: "1001"}, Action{Module: "host", Command: "status"}); err == nil {
		t.Fatalf("expected deny")
	}
}

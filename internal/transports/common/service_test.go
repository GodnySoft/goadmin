package common

import "testing"

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

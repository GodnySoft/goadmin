package common

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"goadmin/internal/storage"
)

// AuditSink записывает аудиторные события.
type AuditSink interface {
	Write(ctx context.Context, ev storage.AuditEvent) error
}

func newRequestID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("req-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func buildAuditPayload(module, command string, args []string) []byte {
	payload, _ := json.Marshal(map[string]interface{}{
		"module":  module,
		"command": command,
		"args":    args,
	})
	return payload
}

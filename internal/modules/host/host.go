package host

import (
	"context"
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"

	"goadmin/internal/core"
)

// Module предоставляет базовые метрики узла.
type Module struct{}

func (m *Module) Name() string { return "host" }

func (m *Module) Init(ctx context.Context) error { //nolint:revive // инициализация пока тривиальна
	return nil
}

func (m *Module) Execute(ctx context.Context, cmd string, args []string) (core.Response, error) {
	switch cmd {
	case "status":
		return m.status(ctx)
	default:
		return core.Response{Status: "error", ErrorCode: "unknown_command"}, fmt.Errorf("command %s not supported", cmd)
	}
}

func (m *Module) status(ctx context.Context) (core.Response, error) {
	hInfo, err := host.InfoWithContext(ctx)
	if err != nil {
		return core.Response{Status: "error", ErrorCode: "host_info_failed"}, fmt.Errorf("host info: %w", err)
	}
	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return core.Response{Status: "error", ErrorCode: "mem_info_failed"}, fmt.Errorf("memory info: %w", err)
	}
	ld, err := load.AvgWithContext(ctx)
	if err != nil {
		return core.Response{Status: "error", ErrorCode: "load_info_failed"}, fmt.Errorf("load info: %w", err)
	}
	resp := map[string]interface{}{
		"hostname":     hInfo.Hostname,
		"platform":     hInfo.Platform,
		"platformVer":  hInfo.PlatformVersion,
		"kernel":       hInfo.KernelVersion,
		"uptime_sec":   hInfo.Uptime,
		"boot_time":    time.Unix(int64(hInfo.BootTime), 0).UTC().Format(time.RFC3339),
		"mem_total":    vm.Total,
		"mem_used":     vm.Used,
		"mem_used_pct": vm.UsedPercent,
		"load1":        ld.Load1,
		"load5":        ld.Load5,
		"load15":       ld.Load15,
	}
	return core.Response{Status: "ok", Data: resp}, nil
}

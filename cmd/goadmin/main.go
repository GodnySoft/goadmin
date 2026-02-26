package main

import (
	"context"
	"log"
	"os"

	"goadmin/internal/core"
	"goadmin/internal/modules/host"
	"goadmin/internal/transports/cli"
	"goadmin/pkg/logger"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
)

func main() {
	lg := logger.New()

	ctx := context.Background()
	r := core.NewRegistry()
	if err := r.Register(ctx, &host.Module{}); err != nil {
		log.Fatalf("register host module: %v", err)
	}

	v := buildVersion()
	root := cli.New(r, v)
	if err := root.ExecuteContext(ctx); err != nil {
		lg.Error("command failed", "err", err)
		os.Exit(1)
	}
}

func buildVersion() string {
	v := version
	if commit != "" {
		v += " (" + commit + ")"
	}
	if date != "" {
		v += " " + date
	}
	return v
}

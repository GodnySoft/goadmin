package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"goadmin/internal/app"
	"goadmin/internal/config"
	"goadmin/internal/core"
)

// New создает корневую CLI-команду.
func New(registry *core.Registry, version string) *cobra.Command {
	root := &cobra.Command{
		Use:   "goadmin",
		Short: "Инфраструктурный агент goadmin",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	var cfgPath string
	root.PersistentFlags().StringVar(&cfgPath, "config", "", "путь к config.yaml")

	root.AddCommand(newVersionCmd(version))
	root.AddCommand(newHostStatusCmd(registry))
	root.AddCommand(newServeCmd(&cfgPath))

	return root
}

func newVersionCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Показать версию",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "%s\n", version)
		},
	}
}

func newHostStatusCmd(registry *core.Registry) *cobra.Command {
	return &cobra.Command{
		Use:   "host status",
		Short: "Показать состояние узла",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Second)
			defer cancel()

			resp, err := registry.Execute(ctx, "host", "status", nil)
			if err != nil {
				return err
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(resp)
		},
	}
}

func newServeCmd(cfgPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Запуск агента в режиме демона",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(*cfgPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			ctx, cancel := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
			defer cancel()

			application, err := app.NewApp(ctx, cfg)
			if err != nil {
				return err
			}
			defer application.Close()

			fmt.Fprintln(cmd.OutOrStdout(), "goadmin serve started")
			if err := application.Serve(ctx); err != nil && ctx.Err() == nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "goadmin serve stopped")
			return nil
		},
	}
}

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

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

	root.AddCommand(newVersionCmd(version))
	root.AddCommand(newHostStatusCmd(registry))

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

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/bootstrap"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/platform/config"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/scheduler"
	"github.com/spf13/cobra"
)

func main() {
	if err := newRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:          "schedule",
		Short:        "Inspect and run application schedules",
		SilenceUsage: true,
	}
	root.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List registered schedules and their next run time",
			RunE: func(cmd *cobra.Command, _ []string) error {
				cfg, registry, err := loadRegistry()
				if err != nil {
					return err
				}
				now := time.Now()
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Default timezone: %s\n", cfg.Scheduler.Timezone); err != nil {
					return err
				}
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), "NAME\tCRON\tTIMEZONE\tQUEUE\tNEXT RUN"); err != nil {
					return err
				}
				for _, definition := range registry.Definitions() {
					if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\t%s\n",
						definition.Name,
						definition.Cron,
						definition.Timezone,
						definition.DispatchOptions.Queue,
						registry.Next(definition, now).Format(time.RFC3339),
					); err != nil {
						return err
					}
				}
				return nil
			},
		},
		&cobra.Command{
			Use:   "run",
			Short: "Enqueue jobs due in the current minute",
			RunE: func(cmd *cobra.Command, _ []string) error {
				cfg, registry, err := loadRegistry()
				if err != nil {
					return err
				}
				dispatcher := bootstrap.Dispatcher(cfg)
				if closer, ok := dispatcher.(interface{ Close() error }); ok {
					defer func() { _ = closer.Close() }()
				}
				if err := scheduler.NewRunner(registry, dispatcher).Run(cmd.Context(), time.Now()); err != nil {
					return err
				}
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "Schedule tick completed")
				return err
			},
		},
	)
	return root
}

func loadRegistry() (*config.Config, *scheduler.Registry, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, err
	}
	registry, err := bootstrap.ScheduleRegistry(cfg)
	return cfg, registry, err
}

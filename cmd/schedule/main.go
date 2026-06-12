package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/bootstrap"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/config"
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
				fmt.Fprintf(cmd.OutOrStdout(), "Default timezone: %s\n", cfg.Scheduler.Timezone)
				fmt.Fprintln(cmd.OutOrStdout(), "NAME\tCRON\tTIMEZONE\tQUEUE\tNEXT RUN")
				for _, definition := range registry.Definitions() {
					fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\t%s\n",
						definition.Name,
						definition.Cron,
						definition.Timezone,
						definition.DispatchOptions.Queue,
						registry.Next(definition, now).Format(time.RFC3339),
					)
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
				dispatcher, err := bootstrap.Dispatcher(cfg)
				if err != nil {
					return err
				}
				if closer, ok := dispatcher.(interface{ Close() error }); ok {
					defer closer.Close()
				}
				if err := scheduler.NewRunner(registry, dispatcher).Run(cmd.Context(), time.Now()); err != nil {
					return err
				}
				log.Println("Schedule tick completed")
				return nil
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

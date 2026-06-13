package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	mysqladapter "github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/mysql"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/bootstrap"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/config"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/scheduler"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
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
				var db *gorm.DB
				if cfg.Queue.Driver == config.QueueDriverDatabase {
					db, err = mysqladapter.Open(cmd.Context(), cfg.Database.DSN, slog.Default())
					if err != nil {
						return err
					}
					defer func() { _ = mysqladapter.Close(db) }()
				}
				dispatcher, err := bootstrap.Dispatcher(cfg, db)
				if err != nil {
					return err
				}
				if closer, ok := dispatcher.(interface{ Close() error }); ok {
					defer func() { _ = closer.Close() }()
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

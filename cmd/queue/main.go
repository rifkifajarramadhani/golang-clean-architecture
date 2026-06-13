package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/jobs"
	mysqladapter "github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/mysql"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/bootstrap"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/config"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/queue"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

func main() {
	if err := newRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	return newRootCommandWithInspector(newInspector)
}

type inspectorFactory func() (queue.Inspector, error)

func newRootCommandWithInspector(factory inspectorFactory) *cobra.Command {
	root := &cobra.Command{Use: "queue", Short: "Inspect and operate application queues", SilenceUsage: true}
	root.AddCommand(statusCommand(factory), failedCommand(factory), retryCommand(factory), deleteCommand(factory), dispatchDemoCommand())
	return root
}

func statusCommand(factory inspectorFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show queue statistics",
		RunE: func(cmd *cobra.Command, _ []string) error {
			inspector, err := factory()
			if err != nil {
				return err
			}
			defer func() { _ = inspector.Close() }()
			queues, err := inspector.Queues(cmd.Context())
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), "QUEUE\tPENDING\tACTIVE\tSCHEDULED\tRETRY\tFAILED\tPROCESSED"); err != nil {
				return err
			}
			for _, name := range queues {
				info, err := inspector.Stats(cmd.Context(), name)
				if err != nil {
					return err
				}
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%d\t%d\t%d\t%d\t%d\t%d\n",
					name, info.Pending, info.Active, info.Scheduled, info.Retry, info.Failed, info.Processed); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func failedCommand(factory inspectorFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "failed",
		Short: "List archived jobs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			inspector, err := factory()
			if err != nil {
				return err
			}
			defer func() { _ = inspector.Close() }()
			queues, err := inspector.Queues(cmd.Context())
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), "ID\tQUEUE\tTYPE\tRETRIES\tFAILED AT\tERROR"); err != nil {
				return err
			}
			for _, name := range queues {
				tasks, err := inspector.Failed(cmd.Context(), name, 100)
				if err != nil {
					return err
				}
				for _, task := range tasks {
					if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%d/%d\t%s\t%s\n",
						task.ID, task.Queue, task.Type, task.Retried, task.MaxRetry,
						task.LastFailedAt.Format("2006-01-02T15:04:05Z07:00"), task.LastError); err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
}

func retryCommand(factory inspectorFactory) *cobra.Command {
	return operateArchivedCommand(factory, "retry <id|all>", "Retry archived jobs", func(ctxCommand *cobra.Command, inspector queue.Inspector, queueName, id string) (int, error) {
		return inspector.Retry(ctxCommand.Context(), queueName, id)
	})
}

func deleteCommand(factory inspectorFactory) *cobra.Command {
	return operateArchivedCommand(factory, "delete <id|all>", "Delete archived jobs", func(ctxCommand *cobra.Command, inspector queue.Inspector, queueName, id string) (int, error) {
		return inspector.Delete(ctxCommand.Context(), queueName, id)
	})
}

func operateArchivedCommand(factory inspectorFactory, use, short string, operation func(*cobra.Command, queue.Inspector, string, string) (int, error)) *cobra.Command {
	var queueName string
	command := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			inspector, err := factory()
			if err != nil {
				return err
			}
			defer func() { _ = inspector.Close() }()
			id := args[0]
			if id != "all" && queueName == "" {
				return fmt.Errorf("--queue is required when operating on one job")
			}
			queues := []string{queueName}
			if id == "all" && queueName == "" {
				queues, err = inspector.Queues(cmd.Context())
				if err != nil {
					return err
				}
			}
			total := 0
			for _, name := range queues {
				count, err := operation(cmd, inspector, name, id)
				if err != nil {
					return err
				}
				total += count
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Affected jobs: %d\n", total)
			return err
		},
	}
	command.Flags().StringVar(&queueName, "queue", "", "Queue name")
	return command
}

func dispatchDemoCommand() *cobra.Command {
	var message string
	command := &cobra.Command{
		Use:   "dispatch-demo",
		Short: "Dispatch a demo logging job",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			db, err := openQueueDB(cmd.Context(), cfg)
			if err != nil {
				return err
			}
			if db != nil {
				defer func() { _ = mysqladapter.Close(db) }()
			}
			dispatcher, err := bootstrap.Dispatcher(cfg, db)
			if err != nil {
				return err
			}
			if closer, ok := dispatcher.(interface{ Close() error }); ok {
				defer func() { _ = closer.Close() }()
			}
			info, err := dispatcher.Dispatch(cmd.Context(), jobs.DemoLog{Message: message}, queue.DispatchOptions{Queue: "default", MaxRetry: 3})
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Dispatched job %s to %s\n", info.ID, info.Queue)
			return err
		},
	}
	command.Flags().StringVar(&message, "message", "Hello from the queue", "Demo log message")
	return command
}

func newInspector() (queue.Inspector, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	db, err := openQueueDB(context.Background(), cfg)
	if err != nil {
		return nil, err
	}
	inspector, err := bootstrap.Inspector(cfg, db)
	if err != nil {
		_ = mysqladapter.Close(db)
		return nil, err
	}
	return &managedInspector{Inspector: inspector, db: db}, nil
}

type managedInspector struct {
	queue.Inspector
	db *gorm.DB
}

func (i *managedInspector) Close() error {
	if err := i.Inspector.Close(); err != nil {
		return err
	}
	return mysqladapter.Close(i.db)
}

func openQueueDB(ctx context.Context, cfg *config.Config) (*gorm.DB, error) {
	if cfg.Queue.Driver != config.QueueDriverDatabase {
		return nil, nil
	}
	return mysqladapter.Open(ctx, cfg.Database.DSN, slog.Default())
}

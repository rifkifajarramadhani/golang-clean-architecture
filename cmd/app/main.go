package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	if err := newRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:          "app",
		Short:        "Run and operate the service in any of its roles",
		SilenceUsage: true,
	}
	root.AddCommand(
		serveCommand(),
		workerCommand(),
		schedulerCommand(),
		queueCommand(),
		scheduleCommand(),
	)
	return root
}

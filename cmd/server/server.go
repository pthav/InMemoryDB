package server

import (
	"github.com/spf13/cobra"
)

func NewServerCmd() *cobra.Command {
	// serverCmd represents the base command when called without any subcommands
	var serverCmd = &cobra.Command{
		Use:   "server",
		Short: "Command line interface for InMemoryDB",
		Long:  `This is a command defines subcommands for InMemoryDB instances`,
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	serverCmd.AddCommand(newServeCmd())

	return serverCmd
}

func init() {
}

package cmd

import (
	"github.com/pthav/InMemoryDB/cmd/server"
	"os"

	"github.com/pthav/InMemoryDB/cmd/endpoint"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	// rootCmd represents the base command when called without any subcommands
	var rootCmd = &cobra.Command{
		Use:   "InMemoryDB",
		Short: "An in memory database that can be served",
		Long: `After the database has been served through the CLI,
it is possible to use the CLI in another terminal
to send requests to the already served database.`,
		Run: func(cmd *cobra.Command, args []string) {},
	}
	rootCmd.AddCommand(endpoint.NewEndpointsCmd())
	rootCmd.AddCommand(server.NewServerCmd())

	return rootCmd
}

func Execute() {
	err := NewRootCmd().Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
}

package server

import (
	"github.com/spf13/cobra"
)

// Common flags for child commands
var url string

// ServerCmd represents the base command when called without any subcommands
var ServerCmd = &cobra.Command{
	Use:   "server",
	Short: "Command line interface for InMemoryDB",
	Long:  `This is a command defines subcommands for InMemoryDB instances`,
	Run:   func(cmd *cobra.Command, args []string) {},
}

func init() {
	ServerCmd.AddCommand(serveCmd)
}

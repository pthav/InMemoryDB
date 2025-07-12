package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "InMemoryDB",
	Short: "An in memory database that can be served",
	Long: `After the database has been served through the CLI,
it is possible to use the CLI in another terminal
to send requests to the already served database.`,
	Run: func(cmd *cobra.Command, args []string) {},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
}

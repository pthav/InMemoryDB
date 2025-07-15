package endpoint

import (
	"fmt"

	"github.com/spf13/cobra"
)

// getTTLCmd represents the getTtl command
var getTTLCmd = &cobra.Command{
	Use:   "getTTL",
	Short: "",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("getTTL called")
		return nil
	},
}

func init() {
}

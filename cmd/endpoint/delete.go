package endpoint

import (
	"fmt"
	"github.com/spf13/cobra"
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a key and its associated value.",
	Long: `The key must be provided in order to delete the key value pair. The returned response code is printed
to the console. delete -k=hello -u='localhost:8080'' will send a delete request for the key 'hello' to a server on port 8080.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Send request
		url := fmt.Sprintf("%v/v1/keys/%v", rootURL, key)
		_, status, err := getResponse("DELETE", url, nil)
		if err != nil {
			return err
		}

		response := StatusPlusErrorResponse{Status: status}

		return outputResponse(cmd, response)
	},
}

func init() {
	deleteCmd.Flags().StringVarP(&key, "key", "k", "", "The key to delete in the database")
	err := deleteCmd.MarkFlagRequired("key")
	if err != nil {
		return
	}
}

package endpoint

import (
	"fmt"
	"github.com/spf13/cobra"
)

func newDeleteCmd(o *Options) *cobra.Command {
	// deleteCmd will delete a key value pair from the database
	var deleteCmd = &cobra.Command{
		Use:   "delete",
		Short: "Delete a key and its associated value.",
		Long: `The key must be provided in order to delete the key value pair. The returned response code is printed
to the console. delete -k=hello -u='localhost:8080'' will send a delete request for the key 'hello' to a server on port 8080.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Send request
			var response StatusPlusErrorResponse
			url := fmt.Sprintf("%v/v1/keys/%v", o.rootURL, o.key)
			status, err := getResponse("DELETE", url, nil, &response)
			if err != nil {
				return err
			}
			response.Status = status

			return outputResponse(cmd, response)
		},
	}

	deleteCmd.Flags().StringVarP(&o.key, "key", "k", "", "The key to delete in the database")
	_ = deleteCmd.MarkFlagRequired("key")

	return deleteCmd
}

func init() {
}

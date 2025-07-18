package endpoint

import (
	"fmt"
	"github.com/spf13/cobra"
)

func newPutCmd(o *Options) *cobra.Command {
	// putCmd represents the put command
	var putCmd = &cobra.Command{
		Use:   "put",
		Short: "Put a key value pair into the database",
		Long: `Put will update the key if it exists in the database or create a new one and attach the passed value to it.
The value and key are required for the put request. The response status code is printed to the console. 
put -k=hello -v=world -p=8080 will put the key value pair (hello,world) into the database listening on port 8080.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create request body
			requestBody := struct {
				Value string `json:"value"`
			}{
				Value: o.value,
			}

			// Send request
			var response StatusPlusErrorResponse
			url := fmt.Sprintf("%v/v1/keys/%v", o.rootURL, o.key)
			status, err := getResponse("PUT", url, requestBody, &response)
			if err != nil {
				return err
			}
			response.Status = status

			return outputResponse(cmd, response)
		},
	}

	putCmd.Flags().StringVarP(&o.key, "key", "k", "", "The key to put into the database")
	putCmd.Flags().StringVarP(&o.value, "value", "v", "", "The value to put into the database")
	_ = putCmd.MarkFlagRequired("key")
	_ = putCmd.MarkFlagRequired("value")

	return putCmd
}

func init() {
}

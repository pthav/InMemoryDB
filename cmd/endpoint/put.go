package endpoint

import (
	"fmt"
	"github.com/spf13/cobra"
)

type httpPutRequest struct {
	Value string `json:"value"`
	Ttl   *int64 `json:"ttl"`
}

func newPutCmd(o *options) *cobra.Command {
	// putCmd puts a key value pair to the database
	var putCmd = &cobra.Command{
		Use:   "put",
		Short: "Put a key value pair into the database",
		Long: `Put will update the key if it exists in the database or create a new one and attach the passed value to it.
The value and key are required for the put request. The response status code is printed to the console. 
put -k=hello -v=world -p=8080 will put the key value pair (hello,world) into the database listening on port 8080.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create request body
			requestBody := httpPutRequest{
				Value: o.value,
			}

			if cmd.Flags().Changed("ttl") {
				ttl := int64(o.ttl)
				requestBody.Ttl = &ttl
			}

			// Send request
			var response statusPlusErrorResponse
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
	putCmd.Flags().IntVar(&o.ttl, "ttl", 0, "The ttl to post to the database")
	_ = putCmd.MarkFlagRequired("key")
	_ = putCmd.MarkFlagRequired("value")

	return putCmd
}

func init() {
}

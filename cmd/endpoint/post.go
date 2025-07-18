package endpoint

import (
	"fmt"
	"github.com/spf13/cobra"
)

type HTTPPostResponse struct {
	Status int    `json:"status"`
	Key    string `json:"key"`
	Error  string `json:"error"`
}

func newPostCmd(o *Options) *cobra.Command {
	// postCmd represents the post command
	var postCmd = &cobra.Command{
		Use:   "post",
		Short: "Post a value to the database",
		Long: `The value must be provided in order to post the value to the database. The response body alongside a 
status code are printed to the console. The response body includes the key associated with the posted value.
post -v=value -p=8080 will send a post request to the server on port 8080.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create request body
			requestBody := struct {
				Value string `json:"value"`
			}{
				Value: o.value,
			}

			// Send request
			url := fmt.Sprintf("%v/v1/keys", o.rootURL)
			_, status, err := getResponse("POST", url, requestBody)
			if err != nil {
				return err
			}

			response := StatusPlusErrorResponse{Status: status}

			return outputResponse(cmd, response)
		},
	}

	postCmd.Flags().StringVarP(&o.value, "value", "v", "", "The value to post to the database")
	_ = postCmd.MarkFlagRequired("value")

	return postCmd
}

func init() {
}

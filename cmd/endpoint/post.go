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

type HTTPPostRequest struct {
	Value string `json:"value"`
	Ttl   *int64 `json:"ttl"`
}

func newPostCmd(o *Options) *cobra.Command {
	// postCmd posts a value to the database
	var postCmd = &cobra.Command{
		Use:   "post",
		Short: "Post a value to the database",
		Long: `The value must be provided in order to post the value to the database. The response body alongside a 
status code are printed to the console. The response body includes the key associated with the posted value.
post -v=value -p=8080 will send a post request to the server on port 8080.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create request body
			requestBody := HTTPPostRequest{
				Value: o.value,
			}

			if cmd.Flags().Changed("ttl") {
				ttl := int64(o.ttl)
				requestBody.Ttl = &ttl
			}

			// Send request
			var response HTTPPostResponse
			url := fmt.Sprintf("%v/v1/keys", o.rootURL)
			status, err := getResponse("POST", url, requestBody, &response)
			if err != nil {
				return err
			}
			response.Status = status

			return outputResponse(cmd, response)
		},
	}

	postCmd.Flags().StringVarP(&o.value, "value", "v", "", "The value to post to the database")
	postCmd.Flags().IntVar(&o.ttl, "ttl", 0, "The ttl to post to the database")
	_ = postCmd.MarkFlagRequired("value")

	return postCmd
}

func init() {
}

package endpoint

import (
	"fmt"
	"github.com/spf13/cobra"
)

type HTTPGetResponse struct {
	Status int    `json:"status"`
	Key    string `json:"key"`
	Value  string `json:"value"`
	Error  string `json:"error"`
}

func newGetCmd(o *Options) *cobra.Command {
	// getCmd gets a key value pair from the database
	var getCmd = &cobra.Command{
		Use:   "get",
		Short: "Get a key value pair.",
		Long: `In order to get a stored key value pair from the database you must provide the key as a parameter.
The returned response is printed to the console as json with the status code. For example, 
get -k=hello -u='localhost:8080' will return the value associated with the hello key in the database listening
on port 8080.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Send request
			var response HTTPGetResponse
			url := fmt.Sprintf("%v/v1/keys/%s", o.rootURL, o.key)
			status, err := getResponse("GET", url, nil, &response)
			if err != nil {
				return err
			}
			response.Status = status

			return outputResponse(cmd, response)
		},
	}

	getCmd.Flags().StringVarP(&o.key, "key", "k", "", "The key to access in the database")
	_ = getCmd.MarkFlagRequired("key")

	return getCmd
}

func init() {
}

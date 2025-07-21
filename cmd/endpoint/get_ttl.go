package endpoint

import (
	"fmt"
	"github.com/spf13/cobra"
)

type HTTPGetTTLResponse struct {
	Status int    `json:"status"`
	Key    string `json:"key"`
	TTL    *int64 `json:"ttl"`
	Error  string `json:"error"`
}

func newGetTTLCmd(o *Options) *cobra.Command {
	// getTTLCmd gets a key and its TTL from the database
	var getTTLCmd = &cobra.Command{
		Use:   "getTTL",
		Short: "Get the remaining TTL for a key",
		Long: `This command fetches the remaining TTL in seconds for a a key value pair. getTTL -k=hello will get the
remaining TTL for key 'hello'. The returned TTL will be null if it is a non-expiring key value pair."`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Send request
			var response HTTPGetTTLResponse
			url := fmt.Sprintf("%v/v1/ttl/%s", o.rootURL, o.key)
			status, err := getResponse("GET", url, nil, &response)
			if err != nil {
				return err
			}
			response.Status = status

			return outputResponse(cmd, response)
		},
	}

	getTTLCmd.Flags().StringVarP(&o.key, "key", "k", "", "The key to access in the database")
	_ = getTTLCmd.MarkFlagRequired("key")

	return getTTLCmd
}

func init() {
}

package endpoint

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
)

type HTTPGetTTLResponse struct {
	Status int    `json:"status"`
	Key    string `json:"key"`
	TTL    *int64 `json:"ttl"`
	Error  string `json:"error"`
}

// getTTLCmd represents the getTtl command
var getTTLCmd = &cobra.Command{
	Use:   "getTTL",
	Short: "Get the remaining TTL for a key",
	Long: `This command fetches the remaining TTL in seconds for a a key value pair. getTTL -k=hello will get the
remaining TTL for key 'hello'. The returned TTL will be null if it is a non-expiring key value pair."`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Send request
		url := fmt.Sprintf("%v/v1/ttl/%s", rootURL, key)

		body, status, err := getResponse("GET", url, nil)
		if err != nil {
			return err
		}

		// Read response body
		var response HTTPGetTTLResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			return errors.New("error decoding response from server")
		}
		response.Status = status

		return outputResponse(cmd, response)
	},
}

func init() {
	getTTLCmd.Flags().StringVarP(&key, "key", "k", "", "The key to access in the database")
	err := getCmd.MarkFlagRequired("key")
	if err != nil {
		return
	}
}

package endpoint

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
)

// getTTLCmd represents the getTtl command
var getTTLCmd = &cobra.Command{
	Use:   "getTTL",
	Short: "Get the remaining TTL for a key",
	Long: `This command fetches the remaining TTL in seconds for a a key value pair. getTTL -k=hello will get the
remaining TTL for key 'hello'. The returned TTL will be null if it is a non-expiring key value pair."`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Send Request
		resp, err := http.Get(fmt.Sprintf("%v/v1/ttl/%s", url, key))
		if err != nil {
			return err
		}

		defer resp.Body.Close()

		// Read response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.New("error reading response from server")
		}

		if resp.StatusCode >= 400 {
			fmt.Println("Status code:", resp.StatusCode)
			fmt.Println("Response body:", string(body))
			return nil
		}

		var out bytes.Buffer
		err = json.Indent(&out, body, "", "  ")
		if err != nil {
			fmt.Println("Invalid JSON:", string(body))
		} else {
			fmt.Println("Status code:", resp.StatusCode)
			fmt.Println(out.String())
		}

		return nil
	},
}

func init() {
	getTTLCmd.Flags().StringVarP(&key, "key", "k", "", "The key to access in the database")
	err := getCmd.MarkFlagRequired("key")
	if err != nil {
		return
	}
}

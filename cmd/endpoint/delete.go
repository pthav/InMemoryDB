package endpoint

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"

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
		url := fmt.Sprintf("%v/v1/keys/%v", url, key)
		req, err := http.NewRequest("DELETE", url, io.NopCloser(bytes.NewBufferString("")))
		if err != nil {
			return err
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}

		defer resp.Body.Close()

		// Read response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.New("error reading response from server")
		}

		fmt.Println("Status code:", resp.StatusCode)
		if resp.StatusCode >= 400 {
			fmt.Println("Response body:", string(body))
		}

		return nil
	},
}

func init() {
	deleteCmd.Flags().StringVarP(&key, "key", "k", "", "The key to delete in the database")
	err := deleteCmd.MarkFlagRequired("key")
	if err != nil {
		return
	}
}

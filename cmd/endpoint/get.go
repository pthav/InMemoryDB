package endpoint

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"net/http"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a key value pair.",
	Long: `In order to get a stored key value pair from the database you must provide the key as a parameter.
The returned response is printed to the console as json with the status code. For example, get -k=hello will return the 
value associated with the hello key in the database.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Send Request
		resp, err := http.Get(fmt.Sprintf("http://localhost:%v/v1/keys/%s", port, key))
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

		err = resp.Body.Close()
		if err != nil {
			return errors.New("error closing response body")
		}

		return nil
	},
}

func init() {
	getCmd.Flags().IntVarP(&port, "port", "p", 8080, "The port to listen on.")
	getCmd.Flags().StringVarP(&key, "key", "k", "", "The key to access in the database")
	err := getCmd.MarkFlagRequired("key")
	if err != nil {
		return
	}
}

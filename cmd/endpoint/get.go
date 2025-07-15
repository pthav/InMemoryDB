package endpoint

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"net/http"
)

type HTTPGetResponse struct {
	Status int    `json:"status"`
	Key    string `json:"key"`
	Value  string `json:"value"`
	Error  string `json:"error"`
}

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a key value pair.",
	Long: `In order to get a stored key value pair from the database you must provide the key as a parameter.
The returned response is printed to the console as json with the status code. For example, 
get -k=hello -u='localhost:8080' will return the value associated with the hello key in the database listening
on port 8080.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Send Request
		resp, err := http.Get(fmt.Sprintf("%v/v1/keys/%s", url, key))
		if err != nil {
			return errors.New(fmt.Sprintf("error creating request: %v", err))
		}

		defer resp.Body.Close()

		// Read response body
		var response HTTPGetResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			return errors.New("error decoding response from server")
		}
		response.Status = resp.StatusCode

		out, err := json.MarshalIndent(response, "", "\t")
		if err != nil {
			return errors.New(fmt.Sprintf("error marshalling response from server: %v", err))
		} else {
			_, err := cmd.OutOrStdout().Write(out)
			if err != nil {
				return err
			}
		}

		err = resp.Body.Close()
		if err != nil {
			return errors.New("error closing response body")
		}

		return nil
	},
}

func init() {
	getCmd.Flags().StringVarP(&key, "key", "k", "", "The key to access in the database")
	err := getCmd.MarkFlagRequired("key")
	if err != nil {
		return
	}
}

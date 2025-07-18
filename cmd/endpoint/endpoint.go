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

// outputResponse is a helper function for outputting JSON to a command's out file and returning an error if there is
// one.
func outputResponse(cmd *cobra.Command, response any) error {
	out, err := json.MarshalIndent(response, "", "\t")
	if err != nil {
		return errors.New(fmt.Sprintf("error marshalling response: %v", err))
	}

	_, err = cmd.OutOrStdout().Write(out)
	if err != nil {
		return err
	}
	return nil
}

// getResponse is a helper function for sending a request and returning the response body, status, and an error
// if there is any.
func getResponse(method string, url string, requestBody any) ([]byte, int, error) {
	// Create request body
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return []byte{}, 0, errors.New(fmt.Sprintf("error marshalling request body: %v", err))
	}

	// Create the request
	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return []byte{}, 0, errors.New(fmt.Sprintf("error creating request: %v", err))
	}

	// Send the request
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return []byte{}, 0, errors.New(fmt.Sprintf("error sending request: %v", err))
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, 0, errors.New(fmt.Sprintf("error reading response from server: %v", err))
	}

	return body, resp.StatusCode, nil
}

// Generic HTTP method response

type StatusPlusErrorResponse struct {
	Status int    `json:"status"`
	Error  string `json:"error"`
}

// Common flags for child commands
var rootURL string
var key string
var value string
var channel string

// EndpointsCmd represents the base command for endpoint commands
var EndpointsCmd = &cobra.Command{
	Use:   "endpoint",
	Short: "Send requests to a database endpoint",
	Long: `This command contains sub commands for sending requests to the endpoint for an instance
of InMemoryDB. The command endpoint get -k=hello -p=8080 will get the key value pair for the database 
listening on port 8080`,
	Run: func(cmd *cobra.Command, args []string) {},
}

func init() {
	EndpointsCmd.AddCommand(getTTLCmd)
	EndpointsCmd.AddCommand(publishCmd)
	EndpointsCmd.AddCommand(subscribeCmd)
	EndpointsCmd.AddCommand(getCmd)
	EndpointsCmd.AddCommand(deleteCmd)
	EndpointsCmd.AddCommand(putCmd)
	EndpointsCmd.AddCommand(postCmd)

	EndpointsCmd.PersistentFlags().StringVarP(&rootURL, "rootURL", "u", "http://localhost:8080", "The rootURL to use.")
}

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

// getResponse is a helper function for sending a request and returning the status and an error
// if there is any.
func getResponse(method string, url string, requestBody any, response any) (int, error) {
	// Create request body
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return 0, errors.New(fmt.Sprintf("error marshalling request body in getResponse(): %v", err))
	}

	// Create the request
	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return 0, errors.New(fmt.Sprintf("error creating request in getResponse(): %v", err))
	}

	// Send the request
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, errors.New(fmt.Sprintf("error sending request in getResponse(): %v", err))
	}
	defer resp.Body.Close()

	// Read the response
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, errors.New(fmt.Sprintf("error reading response body in getResponse(): %v", err))
	}

	err = json.Unmarshal(data, response)
	if err != nil {
		return 0, errors.New(fmt.Sprintf("error decoding response from server in getResponse(). err: %v, body: %v", err, string(data)))
	}

	return resp.StatusCode, nil
}

// Generic HTTP method response

type StatusPlusErrorResponse struct {
	Status int    `json:"status"` // This isn't output as JSON from the external API it is added after.
	Error  string `json:"error"`
}

// Options defines configuration flags for endpoint and its subcommands.
type Options struct {
	rootURL string
	key     string
	value   string
	channel string
	timeout int
	message string
}

func NewEndpointsCmd() *cobra.Command {
	// endpointsCmd represents the base command for endpoint commands
	var endpointsCmd = &cobra.Command{
		Use:   "endpoint",
		Short: "Send requests to a database endpoint",
		Long: `This command contains sub commands for sending requests to the endpoint for an instance
of InMemoryDB. The command endpoint get -k=hello -p=8080 will get the key value pair for the database 
listening on port 8080`,
		Run: func(cmd *cobra.Command, args []string) {},
	}
	o := Options{}

	endpointsCmd.PersistentFlags().StringVarP(&o.rootURL, "rootURL", "u", "http://localhost:8080", "The rootURL to use.")

	endpointsCmd.AddCommand(newGetTTLCmd(&o))
	endpointsCmd.AddCommand(newPublishCmd(&o))
	endpointsCmd.AddCommand(newSubscribeCmd(&o))
	endpointsCmd.AddCommand(newGetCmd(&o))
	endpointsCmd.AddCommand(newDeleteCmd(&o))
	endpointsCmd.AddCommand(newPutCmd(&o))
	endpointsCmd.AddCommand(newPostCmd(&o))

	return endpointsCmd
}

func init() {
}

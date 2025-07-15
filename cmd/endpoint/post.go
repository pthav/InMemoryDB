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

// postCmd represents the post command
var postCmd = &cobra.Command{
	Use:   "post",
	Short: "Post a value to the database",
	Long: `The value must be provided in order to post the value to the database. The response body alongside a 
status code are printed to the console. The response body includes the key associated with the posted value.
post -v=value -p=8080 will send a post request to the server on port 8080.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create request body
		requestBody := struct {
			Value string `json:"value"`
		}{
			Value: value,
		}

		jsonBody, err := json.Marshal(requestBody)
		if err != nil {
			return err
		}

		// Send request
		url := fmt.Sprintf("http://localhost:%v/v1/keys", port)
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}

		// Read response
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
	postCmd.Flags().IntVarP(&port, "port", "p", 8080, "The port to listen on.")
	postCmd.Flags().StringVarP(&value, "value", "v", "", "The value to post to the database")
	err := postCmd.MarkFlagRequired("value")
	if err != nil {
		return
	}
}

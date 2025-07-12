package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
)

// putCmd represents the put command
var putCmd = &cobra.Command{
	Use:   "put",
	Short: "Put a key value pair into the database",
	Long: `Put will update the key if it exists in the database or create a new one and attach the passed value to it.
The value and key are required for the put request. The response status code is printed to the console. 
put -k=hello -v=world -p=8080 will put the key value pair (hello,world) into the database listening on port 8080.`,
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
		fmt.Println(string(jsonBody))

		// Send request
		url := fmt.Sprintf("http://localhost:%v/v1/keys/%v", port, key)
		req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonBody))
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

		fmt.Println("Status code:", resp.StatusCode)
		if resp.StatusCode >= 400 {
			fmt.Println("Response body:", string(body))
		}

		err = resp.Body.Close()
		if err != nil {
			return errors.New("error closing response body")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(putCmd)

	putCmd.Flags().IntVarP(&port, "port", "p", 8080, "The port to listen on.")
	putCmd.Flags().StringVarP(&key, "key", "k", "", "The key to put into the database")
	putCmd.Flags().StringVarP(&value, "value", "v", "", "The value to put into the database")
	err := putCmd.MarkFlagRequired("key")
	if err != nil {
		return
	}
	err = putCmd.MarkFlagRequired("value")
	if err != nil {
		return
	}
}

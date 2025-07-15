package endpoint

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
)

var message string

// publishCmd represents the publish command
var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish a message to a channel",
	Long: `This command publishes a message to a channel such that all listening subscribers will receive that
message. publish -c=hello -m=world will publish 'world' to the channel 'hello' `,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create request body
		payload := fmt.Sprintf(`{"message": "%v"}`, message)

		// Send Request
		url := fmt.Sprintf("%v/v1/publish/%s", url, channel)
		resp, err := http.Post(url, "application/json", strings.NewReader(payload))
		if err != nil {
			return err
		}

		defer resp.Body.Close()

		fmt.Println("Status code:", resp.StatusCode)

		return nil
	},
}

func init() {
	publishCmd.Flags().StringVarP(&message, "message", "m", "", "The message to publish")
	publishCmd.Flags().StringVarP(&channel, "channel", "c", "", "The channel to post a message to")

	err := postCmd.MarkFlagRequired("message")
	if err != nil {
		return
	}
	err = postCmd.MarkFlagRequired("channel")
	if err != nil {
		return
	}
}

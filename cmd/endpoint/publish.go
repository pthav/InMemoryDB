package endpoint

import (
	"fmt"
	"github.com/spf13/cobra"
)

func newPublishCmd(o *options) *cobra.Command {
	// publishCmd publishes a message to a channel in the database
	var publishCmd = &cobra.Command{
		Use:   "publish",
		Short: "Publish a message to a channel",
		Long: `This command publishes a message to a channel such that all listening subscribers will receive that
message. publish -c=hello -m=world will publish 'world' to the channel 'hello' `,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create request body
			payload := struct {
				Message string `json:"message"`
			}{
				Message: o.message,
			}

			// Send Request
			var response statusPlusErrorResponse
			url := fmt.Sprintf("%v/v1/publish/%s", o.rootURL, o.channel)
			status, err := getResponse("POST", url, payload, &response)
			if err != nil {
				return err
			}
			response.Status = status

			return outputResponse(cmd, response)
		},
	}

	publishCmd.Flags().StringVarP(&o.message, "message", "m", "", "The message to publish")
	publishCmd.Flags().StringVarP(&o.channel, "channel", "c", "", "The channel to post a message to")

	_ = publishCmd.MarkFlagRequired("message")
	_ = publishCmd.MarkFlagRequired("channel")

	return publishCmd
}

func init() {
}

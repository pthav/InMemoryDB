package endpoint

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newSubscribeCmd(o *Options) *cobra.Command {
	// subscribeCmd subscribes to a channel in the database
	var subscribeCmd = &cobra.Command{
		Use:   "subscribe",
		Short: "Subscribe to a channel",
		Long: `Subscribing to a channel allows receival of published messages to that channel. subscribe -c=hello -t=30
will subscribe to channel 'hello' for up to 30 seconds.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create an http request for subscription that will automatically disconnect after the expiration
			client := http.Client{}

			ctx, cancel := context.WithTimeout(cmd.Context(), time.Duration(o.timeout)*time.Second)
			defer cancel()

			url := fmt.Sprintf("%v/v1/subscribe/%s", o.rootURL, o.channel)
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				return err
			}

			resp, err := client.Do(req)
			if err != nil {
				return errors.New(fmt.Sprintf("error sending request to server: %v", err))
			}
			defer resp.Body.Close()

			reader := bufio.NewReader(resp.Body)

			// Get each message
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					// Check if it is an organic error
					if errors.Is(err, context.DeadlineExceeded) || err == io.EOF {
						return nil
					}
					return err
				}

				// Only print valid SSE output
				if strings.HasPrefix(line, "data: ") {
					_, err = cmd.OutOrStdout().Write([]byte(line))
					if err != nil {
						return err
					}
				}
			}
		},
	}

	subscribeCmd.Flags().StringVarP(&o.channel, "channel", "c", "", "The channel to subscribe to")
	subscribeCmd.Flags().IntVarP(&o.timeout, "timeout", "t", 60, "How long to subscribe for")
	_ = subscribeCmd.MarkFlagRequired("channel")

	return subscribeCmd
}

func init() {
}

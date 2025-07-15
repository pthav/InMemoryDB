package endpoint

import (
	"github.com/spf13/cobra"
)

// HTTP method-specific responses

type HTTPPostResponse struct {
	Status string `json:"status"`
	Key    string `json:"key"`
}

type HTTPGetTTLResponse struct {
	Status string `json:"status"`
	Key    string `json:"key"`
	TTL    *int64 `json:"ttl"`
}

type StatusOnlyResponse struct {
	Status string `json:"status"`
}

// Common flags for child commands
var url string
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

	EndpointsCmd.PersistentFlags().StringVarP(&url, "url", "u", "http://localhost:8080", "The url to use.")
}

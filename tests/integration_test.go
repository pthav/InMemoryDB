package tests

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"github.com/pthav/InMemoryDB/cmd"
	"github.com/spf13/cobra"
	"strings"
	"sync"
	"testing"
	"time"
)

// execute is a helper function for executing commands.
func execute(t *testing.T, c *cobra.Command, args ...string) (string, error) {
	t.Helper()

	buf := new(bytes.Buffer)
	c.SetOut(buf)
	c.SetErr(buf)
	c.SetArgs(args)

	err := c.Execute()
	return strings.TrimSpace(buf.String()), err
}

type httpPostResponse struct {
	Status int    `json:"status"`
	Key    string `json:"key"`
	Error  string `json:"error"`
}

type httpGetResponse struct {
	Status int    `json:"status"`
	Key    string `json:"key"`
	Value  string `json:"value"`
	Error  string `json:"error"`
}

type httpGetTTLResponse struct {
	Status int    `json:"status"`
	Key    string `json:"key"`
	TTL    *int64 `json:"ttl"`
	Error  string `json:"error"`
}

type statusPlusErrorResponse struct {
	Status int    `json:"status"`
	Error  string `json:"error"`
}

func TestInMemoryDB_integration_test(t *testing.T) {
	type operations []struct {
		args           []string      // Arguments to be passed to the cli
		postValue      string        // The value posted by a post operation
		postTTL        int64         // The TTL posted by a post operation
		cliShouldError bool          // Whether the cli should itself error
		expected       any           // The expected response
		wait           time.Duration // Time to wait before running operation
	}

	testCases := []struct {
		name       string // Name of the test case
		operations operations
	}{
		{
			name: "Get with key that doesn't exist",
			operations: operations{
				{
					args:           []string{"endpoint", "get", "-k", "hello"},
					cliShouldError: false,
					expected:       httpGetResponse{Status: 404},
				},
			},
		},
		{
			name: "Get with key that exists",
			operations: operations{
				{
					args:           []string{"endpoint", "put", "-k", "hello", "-v", "hello"},
					cliShouldError: false,
					expected:       statusPlusErrorResponse{Status: 201},
				},
				{
					args:           []string{"endpoint", "get", "-k", "hello"},
					cliShouldError: false,
					expected:       httpGetResponse{Status: 200, Key: "hello", Value: "hello"},
				},
			},
		},
		{
			name: "GetTTL with key that doesn't exist",
			operations: operations{
				{
					args:           []string{"endpoint", "getTTL", "-k", "hello"},
					cliShouldError: false,
					expected:       httpGetTTLResponse{Status: 404},
				},
			},
		},
		{
			name: "GetTTL with key that exists",
			operations: operations{
				{
					args:           []string{"endpoint", "put", "-k", "hello", "-v", "hello", "--ttl", "10"},
					cliShouldError: false,
					expected:       statusPlusErrorResponse{Status: 201},
				},
				{
					args:           []string{"endpoint", "getTTL", "-k", "hello"},
					cliShouldError: false,
					expected:       httpGetTTLResponse{Status: 200, Key: "hello", TTL: intToPtr(10)},
				},
			},
		},
		{
			name: "GetTTL with key that exists and null TTL",
			operations: operations{
				{
					args:           []string{"endpoint", "put", "-k", "hello", "-v", "hello"},
					cliShouldError: false,
					expected:       statusPlusErrorResponse{Status: 201},
				},
				{
					args:           []string{"endpoint", "getTTL", "-k", "hello"},
					cliShouldError: false,
					expected:       httpGetTTLResponse{Status: 200, Key: "hello", TTL: nil},
				},
			},
		},
		{
			name: "Get and GetTTL after updating with put",
			operations: operations{
				{
					args:           []string{"endpoint", "put", "-k", "hello", "-v", "hello"},
					cliShouldError: false,
					expected:       statusPlusErrorResponse{Status: 201},
				},
				{
					args:           []string{"endpoint", "put", "-k", "hello", "-v", "update", "--ttl", "10"},
					cliShouldError: false,
					expected:       statusPlusErrorResponse{Status: 200},
				},
				{
					args:           []string{"endpoint", "get", "-k", "hello"},
					cliShouldError: false,
					expected:       httpGetResponse{Status: 200, Key: "hello", Value: "update"},
				},
				{
					args:           []string{"endpoint", "getTTL", "-k", "hello"},
					cliShouldError: false,
					expected:       httpGetTTLResponse{Status: 200, Key: "hello", TTL: intToPtr(10)},
				},
			},
		},
		{
			name: "Get and GetTTL after expiration",
			operations: operations{
				{
					args:           []string{"endpoint", "put", "-k", "hello", "-v", "update", "--ttl", "0"},
					cliShouldError: false,
					expected:       statusPlusErrorResponse{Status: 201},
				},
				{
					args:           []string{"endpoint", "get", "-k", "hello"},
					cliShouldError: false,
					expected:       httpGetResponse{Status: 404},
				},
				{
					args:           []string{"endpoint", "getTTL", "-k", "hello"},
					cliShouldError: false,
					expected:       httpGetTTLResponse{Status: 404},
				},
			},
		},
		{
			name: "Posting",
			operations: operations{
				{
					args:           []string{"endpoint", "post", "-v", "posting"},
					postValue:      "posting",
					cliShouldError: false,
					expected:       httpPostResponse{Status: 201},
				},
				{
					args:           []string{"endpoint", "post", "-v", "posting", "--ttl", "10"},
					postValue:      "posting",
					postTTL:        10,
					cliShouldError: false,
					expected:       httpPostResponse{Status: 201},
				},
			},
		},
		{
			name: "Delete a key that doesn't exist",
			operations: operations{
				{
					args:           []string{"endpoint", "delete", "-k", "hello"},
					cliShouldError: false,
					expected:       statusPlusErrorResponse{Status: 404},
				},
			},
		},
		{
			name: "Delete a key that does exist",
			operations: operations{
				{
					args:           []string{"endpoint", "put", "-k", "hello", "-v", "hello"},
					cliShouldError: false,
					expected:       httpGetResponse{Status: 201},
				},
				{
					args:           []string{"endpoint", "delete", "-k", "hello"},
					cliShouldError: false,
					expected:       statusPlusErrorResponse{Status: 200},
				},
			},
		},
		{
			name: "Invalid args",
			operations: operations{
				{
					args:           []string{"endpoint", "put", "-v", "hello"},
					cliShouldError: true,
				},
				{
					args:           []string{"endpoint", "put", "-k", "hello"},
					cliShouldError: true,
				},
				{
					args:           []string{"endpoint", "delete"},
					cliShouldError: true,
				},
				{
					args:           []string{"endpoint", "get"},
					cliShouldError: true,
				},
				{
					args:           []string{"endpoint", "getTTL"},
					cliShouldError: true,
				},
				{
					args:           []string{"endpoint", "post"},
					cliShouldError: true,
				},
				{
					args:           []string{"endpoint", "publish", "-m", "hello"},
					cliShouldError: true,
				},
				{
					args:           []string{"endpoint", "publish", "-c", "hello"},
					cliShouldError: true,
				},
				{
					args:           []string{"endpoint", "sub", "-m", "hello"},
					cliShouldError: true,
				},
				{
					args:           []string{"endpoint", "sub", "-c", "hello"},
					cliShouldError: true,
				},
				{
					args:           []string{"server", "serve", "--persist"},
					cliShouldError: true,
				},
				{
					args:           []string{"server", "serve", "--persist-file", "blah.json"},
					cliShouldError: true,
				},
				{
					args:           []string{"dogs"},
					cliShouldError: true,
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(1)
			dir := t.TempDir()
			serverStartArgs := []string{"server", "serve",
				"--startup-file", "startup.json",
				"--persist", "-c", "1", "--persist-file", dir + "persist.json",
				"--no-log",
			}
			ctx, cancel := context.WithCancel(context.Background())
			serverCmd := cmd.NewRootCmd()
			serverCmd.SetArgs(serverStartArgs)
			serverCmd.SetContext(ctx)
			go func() {
				defer wg.Done()
				err := serverCmd.ExecuteContext(ctx)
				if err != nil {
					t.Errorf("Error executing server command with context: %v", err)
				}
			}()

			<-time.After(100 * time.Millisecond) // Wait for server to set up

			for i, op := range tt.operations {
				<-time.After(op.wait)
				t.Logf("Running operation %v with args %v", i, op.args)
				out, err := execute(t, cmd.NewRootCmd(), op.args...)

				if op.cliShouldError {
					if err == nil {
						t.Errorf("CLI should have an error")
					}
					continue
				}

				if err != nil {
					t.Errorf("Expected no error from the CLI but got one: %v", err)
				}

				switch op.expected.(type) {
				case httpPostResponse:
					expected := op.expected.(httpPostResponse)
					var result httpPostResponse
					err := json.Unmarshal([]byte(out), &result)
					if err != nil {
						t.Errorf("Error unmarshalling json: %v", err)
					}
					if result.Status != expected.Status {
						t.Fatalf("Expected status to be %v but got %v", expected.Status, result.Status)
					}

					if expected.Status >= 400 && result.Error == "" {
						t.Errorf("Expected error to be non empty but it was empty")
					} else if expected.Status < 400 {
						if result.Key == "" {
							t.Errorf("Expected a key in the response but didn't get one")
						} else {
							// Make sure it was created with the value
							out, err := execute(t, cmd.NewRootCmd(), []string{"endpoint", "get", "-k", result.Key}...)
							if err != nil {
								t.Fatalf("Expected no error from the CLI but got one: %v", err)
							}

							var res httpGetResponse
							err = json.Unmarshal([]byte(out), &res)
							if err != nil {
								t.Fatalf("Error unmarshalling json: %v", err)
							}

							if res.Status != 200 || res.Key != result.Key || res.Value != op.postValue {
								t.Errorf("Expected response to be %v, %v, %v. Instead got %v, %v, %v",
									200, result.Key, op.postValue, res.Status, res.Key, res.Value,
								)
							}

							// Make sure it was created with the TTL
							if op.postTTL != 0 {
								out, err := execute(t, cmd.NewRootCmd(), []string{"endpoint", "getTTL", "-k", result.Key}...)
								if err != nil {
									t.Fatalf("Expected no error from the CLI but got one: %v", err)
								}

								var res httpGetTTLResponse
								err = json.Unmarshal([]byte(out), &res)
								if err != nil {
									t.Fatalf("Error unmarshalling json: %v", err)
								}

								if res.Status != 200 || res.Key != result.Key || *res.TTL < op.postTTL-2 {
									t.Errorf("Expected response to be %v, %v, %v. Instead got %v, %v, %v",
										200, result.Key, op.postTTL, res.Status, res.Key, *res.TTL,
									)
								}
							}
						}
					}
				case httpGetResponse:
					expected := op.expected.(httpGetResponse)
					var result httpGetResponse
					err := json.Unmarshal([]byte(out), &result)
					if err != nil {
						t.Errorf("Error unmarshalling json: %v", err)
					}

					if result.Status != expected.Status {
						t.Fatalf("Expected status to be %v but got %v", expected.Status, result.Status)
					}

					if expected.Status >= 400 && result.Error == "" {
						t.Errorf("Expected error to be non empty but it was empty")
					} else if expected.Status < 400 {
						if result.Key != expected.Key {
							t.Errorf("Expected key to be %v but got %v", expected.Key, result.Key)
						}
						if result.Value != expected.Value {
							t.Errorf("Expected value to be %v but got %v", expected.Value, result.Value)
						}
					}
				case statusPlusErrorResponse:
					expected := op.expected.(statusPlusErrorResponse)
					var result statusPlusErrorResponse
					err := json.Unmarshal([]byte(out), &result)
					if err != nil {
						t.Errorf("Error unmarshalling json: %v", err)
					}

					if expected.Status >= 400 && result.Error == "" {
						t.Errorf("Expected response to be %v but got %v", op.expected, result)
					}
				case httpGetTTLResponse:
					expected := op.expected.(httpGetTTLResponse)
					var result httpGetTTLResponse
					err := json.Unmarshal([]byte(out), &result)
					if err != nil {
						t.Errorf("Error unmarshalling json: %v", err)
					}
					if result.Status != expected.Status {
						t.Fatalf("Expected status to be %v but got %v", expected.Status, result.Status)
					}

					if expected.Status >= 400 && result.Error == "" {
						t.Errorf("Expected error to be non empty but it was empty")
					} else if expected.Status < 400 {
						if result.Key != expected.Key {
							t.Errorf("Expected key to be %v but got %v", expected.Key, result.Key)
						}
						if expected.TTL == nil && result.TTL != nil {
							t.Fatalf("Expected TTL to be nil but got %v", result.TTL)
						}
						if expected.TTL != nil && result.TTL != nil && *result.TTL < *expected.TTL-2 {
							t.Errorf("Expected value to be %v but got %v", *expected.TTL, *result.TTL)
						}
					}
				}
			}
			cancel()
			wg.Wait()
		})
	}
}

func TestInMemoryDB_integration_pubSub_test(t *testing.T) {
	type subscriber struct {
		channel  string
		expected []string
		expire   string
	}

	type publisher struct {
		channel string
		message string
		wait    time.Duration
	}

	tests := []struct {
		name        string
		subscribers []subscriber
		publishers  []publisher
	}{
		{
			name: "One subscriber",
			subscribers: []subscriber{
				{channel: "test", expected: []string{"message1", "message2"}, expire: "1"},
			},
			publishers: []publisher{
				{channel: "test", message: "message1", wait: 60 * time.Millisecond},
				{channel: "test", message: "message2", wait: 80 * time.Millisecond},
			},
		},
		{
			name: "Multiple subscribers",
			subscribers: []subscriber{
				{channel: "test", expected: []string{"message1", "message2"}, expire: "1"},
				{channel: "test", expected: []string{"message1", "message2"}, expire: "1"},
				{channel: "dogs", expected: []string{"message1", "message2", "message3", "message4"}, expire: "1"},
			},
			publishers: []publisher{
				{channel: "test", message: "message1", wait: 60 * time.Millisecond},
				{channel: "test", message: "message2", wait: 80 * time.Millisecond},
				{channel: "dogs", message: "message1", wait: 60 * time.Millisecond},
				{channel: "dogs", message: "message2", wait: 80 * time.Millisecond},
				{channel: "dogs", message: "message3", wait: 100 * time.Millisecond},
				{channel: "dogs", message: "message4", wait: 120 * time.Millisecond},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var wg sync.WaitGroup
			dir := t.TempDir()
			serverStartArgs := []string{"server", "serve",
				"--startup-file", "startup.json",
				"--persist", "-c", "1", "--persist-file", dir + "persist.json",
				"--no-log",
			}
			ctx, cancel := context.WithCancel(context.Background())
			serverCmd := cmd.NewRootCmd()
			serverCmd.SetArgs(serverStartArgs)
			serverCmd.SetContext(ctx)
			go func() {
				err := serverCmd.ExecuteContext(ctx)
				if err != nil {
					t.Errorf("Error executing server command with context: %v", err)
				}
			}()

			<-time.After(100 * time.Millisecond) // Wait for server to set up

			for i, s := range tt.subscribers {
				wg.Add(1)
				go func() {
					defer wg.Done()
					t.Logf("Subscriber %v subscribing to channel %v", i, s.channel)

					args := []string{"endpoint", "subscribe", "-c", s.channel, "-t", s.expire}
					output, err := execute(t, cmd.NewRootCmd(), args...)
					if err != nil {
						t.Error(err)
						return
					}

					// Get each message
					messageCount := 0
					scanner := bufio.NewScanner(strings.NewReader(output))
					for scanner.Scan() {
						line := scanner.Text()
						t.Logf("Subscriber %v has received line %v", i, line)

						// Only check valid SSE output
						if strings.HasPrefix(line, "data: ") {
							if messageCount > len(s.expected) {
								t.Errorf("Too many messages received got %v expected %v", messageCount, len(s.expected))
								break
							}
							msg := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
							if msg != s.expected[messageCount] {
								t.Errorf("For message %v expected %v but got %v", messageCount, s.expected[messageCount], msg)
								break
							}
							messageCount++
						}
					}
					if messageCount != len(s.expected) {
						t.Errorf("Incorrect message count got %v expected %v", messageCount, len(s.expected))
					}
					if err = scanner.Err(); err != nil {
						t.Error(err)
					}
				}()
			}

			// Start each publisher
			for _, p := range tt.publishers {
				wg.Add(1)
				go func() {
					defer wg.Done()
					args := []string{"endpoint", "publish", "-c", p.channel, "-m", p.message}
					<-time.After(p.wait)
					_, err := execute(t, cmd.NewRootCmd(), args...)
					if err != nil {
						t.Errorf("Error executing publish: %v", err)
					}
				}()
			}

			wg.Wait()
			cancel()
		})
	}
}

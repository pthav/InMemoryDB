package cmd

import (
	"InMemoryDB/cmd/endpoint"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type testCase struct {
	name             string   // Test case name
	key              string   // Key for the request
	value            string   // Value for the request
	returnStatus     int      // What the handler should set the status to
	response         any      // The response the handler should return
	writeBadJSON     bool     // Whether the server should write bad JSON
	badURL           bool     // Whether a bad url should be used or not
	shouldError      bool     // Whether the command should return an error
	expectedError    string   // A substring that a returned error should contain
	alternateArgs    []string // Allows test cases to create custom argument slices
	useAlternateArgs bool     // Whether the alternate args should be used
}

// Common test case for testing a bad JSON response from the server
var badJSONTest = testCase{
	name:          "Test bad JSON from server",
	key:           "hello",
	returnStatus:  200,
	writeBadJSON:  true,
	badURL:        false,
	shouldError:   true,
	expectedError: "error decoding response from server",
}

// Common test case for testing a bad server url
var badURLTest = testCase{
	name:          "Test bad url to server",
	key:           "hello",
	writeBadJSON:  false,
	badURL:        true,
	shouldError:   true,
	expectedError: "error sending request",
}

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

// handlerHelper creates and returns a new mux router for the test cases.
func handlerHelper(url string, returnStatus int, response any, badJSON bool) *mux.Router {
	var router *mux.Router
	router = mux.NewRouter()
	router.HandleFunc(url, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(returnStatus)

		// Write badJSON if necessary for the test case
		if badJSON {
			_, err := fmt.Fprint(w, response)
			if err != nil {
				return
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			return
		}
	})
	return router
}

// testHelper will spin up a test server for sending requests to, execute the command, and check outputs.
func testHelper(t *testing.T, tt testCase, url string, args []string) {
	// Spin up mock server
	h := handlerHelper(url, tt.returnStatus, tt.response, tt.writeBadJSON)
	ts := httptest.NewServer(h)
	defer ts.Close()

	if tt.badURL {
		args = append(args, "--bad-url")
	} else {
		args = append(args, ts.URL)
	}

	// Prevent persistence between test cases
	out, err := execute(t, newRootCmd(), args...)

	if (err != nil) != tt.shouldError {
		t.Errorf("expected shouldError(%v), got %v", tt.shouldError, err)
	}

	if tt.shouldError && err != nil && !strings.Contains(err.Error(), tt.expectedError) {
		t.Errorf("expected error to contain %v, got %v", tt.expectedError, err)
	}

	if !tt.shouldError {
		// Type switch to make result the correct type
		var result any
		switch tt.response.(type) {
		case endpoint.HTTPGetResponse:
			result = new(endpoint.HTTPGetResponse)
		case endpoint.HTTPPostResponse:
			result = new(endpoint.HTTPPostResponse)
		case endpoint.HTTPGetTTLResponse:
			result = new(endpoint.HTTPGetTTLResponse)
		case endpoint.StatusPlusErrorResponse:
			result = new(endpoint.StatusPlusErrorResponse)
		}

		err = json.Unmarshal([]byte(out), &result)
		if err != nil {
			t.Error(err)
		}

		// Type switch to make the correct comparison with the reflect package
		switch expected := tt.response.(type) {
		case endpoint.HTTPGetResponse:
			if !reflect.DeepEqual(result, &expected) {
				t.Errorf("got %v\nwant %v", result, &expected)
			}
		case endpoint.HTTPPostResponse:
			if !reflect.DeepEqual(result, &expected) {
				t.Errorf("got %v\nwant %v", result, &expected)
			}
		case endpoint.HTTPGetTTLResponse:
			if !reflect.DeepEqual(result, &expected) {
				t.Errorf("got %v\nwant %v", result, &expected)
			}
		case endpoint.StatusPlusErrorResponse:
			if !reflect.DeepEqual(result, &expected) {
				t.Errorf("got %v\nwant %v", result, &expected)
			}
		}
	}
}

func TestCommand_get(t *testing.T) {
	tests := []testCase{
		{
			name:         "Test forwards response",
			key:          "hello",
			returnStatus: 200,
			response:     endpoint.HTTPGetResponse{Status: 200, Key: "hello", Value: "world", Error: "null"},
			writeBadJSON: false,
			badURL:       false,
			shouldError:  false,
		},
		{
			name:             "Missing the key flag",
			alternateArgs:    []string{"endpoint", "get"},
			useAlternateArgs: true,
			shouldError:      true,
			expectedError:    "required",
		},
		badJSONTest,
		badURLTest,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/v1/keys/{key}"
			args := []string{"endpoint", "get", "-k", tt.key, "-u"}
			if tt.useAlternateArgs {
				testHelper(t, tt, url, tt.alternateArgs)
			} else {
				testHelper(t, tt, url, args)
			}
		})
	}
}

func TestCommand_delete(t *testing.T) {
	tests := []testCase{
		{
			name:         "Test forwards response",
			key:          "hello",
			returnStatus: 200,
			response:     endpoint.StatusPlusErrorResponse{Status: 200, Error: "null"},
			writeBadJSON: false,
			badURL:       false,
			shouldError:  false,
		},
		{
			name:             "Missing the key flag",
			alternateArgs:    []string{"endpoint", "delete"},
			useAlternateArgs: true,
			shouldError:      true,
			expectedError:    "required",
		},
		badURLTest,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/v1/keys/{key}"
			args := []string{"endpoint", "delete", "-k", tt.key, "-u"}
			if tt.useAlternateArgs {
				testHelper(t, tt, url, tt.alternateArgs)
			} else {
				testHelper(t, tt, url, args)
			}
		})
	}
}

func TestCommand_put(t *testing.T) {
	tests := []testCase{
		{
			name:         "Test forwards response",
			key:          "hello",
			value:        "world",
			returnStatus: 200,
			response:     endpoint.StatusPlusErrorResponse{Status: 200, Error: "null"},
			writeBadJSON: false,
			badURL:       false,
			shouldError:  false,
		},
		{
			name:             "Missing the key flag",
			alternateArgs:    []string{"endpoint", "put", "-v", "world"},
			useAlternateArgs: true,
			shouldError:      true,
			expectedError:    "required",
		},
		{
			name:             "Missing the value flag",
			alternateArgs:    []string{"endpoint", "put", "-k", "hello"},
			useAlternateArgs: true,
			shouldError:      true,
			expectedError:    "required",
		},
		badURLTest,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/v1/keys/{key}"
			args := []string{"endpoint", "put", "-k", tt.key, "-v", tt.value, "-u"}
			if tt.useAlternateArgs {
				testHelper(t, tt, url, tt.alternateArgs)
			} else {
				testHelper(t, tt, url, args)
			}
		})
	}
}

func TestCommand_post(t *testing.T) {
	tests := []testCase{
		{
			name:         "Test forwards response",
			returnStatus: 200,
			value:        "world",
			response:     endpoint.HTTPPostResponse{Status: 200, Key: "postKey", Error: "null"},
			writeBadJSON: false,
			badURL:       false,
			shouldError:  false,
		},
		{
			name:             "Missing the value flag",
			alternateArgs:    []string{"endpoint", "post"},
			useAlternateArgs: true,
			shouldError:      true,
			expectedError:    "required",
		},
		badURLTest,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/v1/keys"
			args := []string{"endpoint", "post", "-v", tt.value, "-u"}
			if tt.useAlternateArgs {
				testHelper(t, tt, url, tt.alternateArgs)
			} else {
				testHelper(t, tt, url, args)
			}
		})
	}
}

func TestCommand_getTTL(t *testing.T) {
	intPtr := func(v int64) *int64 {
		return &v
	}

	tests := []testCase{
		{
			name:         "Test forwards response",
			returnStatus: 200,
			key:          "hello",
			response:     endpoint.HTTPGetTTLResponse{Status: 200, Key: "hello", TTL: intPtr(100), Error: "null"},
			writeBadJSON: false,
			badURL:       false,
			shouldError:  false,
		},
		{
			name:             "Missing the key flag",
			alternateArgs:    []string{"endpoint", "getTTL"},
			useAlternateArgs: true,
			shouldError:      true,
			expectedError:    "required",
		},
		badJSONTest,
		badURLTest,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/v1/ttl/{key}"
			args := []string{"endpoint", "getTTL", "-k", tt.key, "-u"}
			if tt.useAlternateArgs {
				testHelper(t, tt, url, tt.alternateArgs)
			} else {
				testHelper(t, tt, url, args)
			}
		})
	}
}

func TestCommand_pubSub(t *testing.T) {
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
		t           testCase
		subscribers []subscriber
		publishers  []publisher
		wait        time.Duration
	}{
		{
			t: testCase{
				name: "One subscriber",
			},
			subscribers: []subscriber{
				{channel: "test", expected: []string{"message1", "message2"}, expire: "1"},
			},
			publishers: []publisher{
				{channel: "test", message: "message1", wait: 10 * time.Millisecond},
				{channel: "test", message: "message2", wait: 20 * time.Millisecond},
			},
			wait: 100 * time.Millisecond,
		},
		{
			t: testCase{
				name: "Multiple subscribers",
			},
			subscribers: []subscriber{
				{channel: "test", expected: []string{"message1", "message2"}, expire: "1"},
				{channel: "test", expected: []string{"message1", "message2"}, expire: "1"},
				{channel: "dogs", expected: []string{"message1", "message2", "message3", "message4"}, expire: "1"},
			},
			publishers: []publisher{
				{channel: "test", message: "message1", wait: 10 * time.Millisecond},
				{channel: "test", message: "message2", wait: 20 * time.Millisecond},
				{channel: "dogs", message: "message1", wait: 10 * time.Millisecond},
				{channel: "dogs", message: "message2", wait: 20 * time.Millisecond},
				{channel: "dogs", message: "message3", wait: 30 * time.Millisecond},
				{channel: "dogs", message: "message4", wait: 40 * time.Millisecond},
			},
			wait: 100 * time.Millisecond,
		},
	}

	type publishRequest struct {
		Message string `json:"message" validate:"required"`
	}

	for _, tt := range tests {
		t.Run(tt.t.name, func(t *testing.T) {
			channels := map[string][]chan string{
				"test": make([]chan string, 2),
				"dogs": make([]chan string, 1),
			}
			var mu sync.RWMutex
			m := mux.NewRouter()
			m.HandleFunc("/v1/subscribe/{channel}", func(w http.ResponseWriter, r *http.Request) {
				vars := mux.Vars(r)
				channel := vars["channel"]

				// Check if SSE is valid for the writer
				flusher, ok := w.(http.Flusher)
				if !ok {
					t.Fatalf("Streaming unsupported")
				}

				// SSE headers
				w.Header().Set("Content-Type", "text/event-stream")
				w.Header().Set("Cache-Control", "no-cache")
				w.Header().Set("Connection", "keep-alive")

				c := make(chan string, 10)

				// Run a go func to remove the subscriber from the channel when they disconnect
				ctx := r.Context()
				go func() {
					<-ctx.Done()
					mu.Lock()
					for i, ch := range channels[channel] {
						if ch == c {
							channels[channel] = append(channels[channel][:i], channels[channel][i+1:]...)
							break
						}
					}
					close(c)
					mu.Unlock()
				}()

				mu.Lock()
				channels[channel] = append(channels[channel], c)
				mu.Unlock()

				for message := range c {
					_, err := fmt.Fprintf(w, "data: %s\n\n", message)
					if err != nil {
						t.Error(err)
					}
					flusher.Flush()
				}
			}).Methods("GET")
			m.HandleFunc("/v1/publish/{channel}", func(w http.ResponseWriter, r *http.Request) {
				vars := mux.Vars(r)
				channel := vars["channel"]

				var pData publishRequest
				if err := json.NewDecoder(r.Body).Decode(&pData); err != nil {
					http.Error(w, "Publish request has bad body", http.StatusBadRequest)
				}

				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"error":"null"}"`))
				if err != nil {
					return
				}

				mu.RLock()
				for _, c := range channels[channel] {
					select {
					case c <- pData.Message:
					default:
						// Drop message if the channel is full
					}
				}
				mu.RUnlock()

			}).Methods("POST")
			ts := httptest.NewServer(m)
			defer ts.Close()

			// Start each subscriber
			var wg sync.WaitGroup
			for i, s := range tt.subscribers {
				wg.Add(1)
				go func() {
					defer wg.Done()
					t.Logf("Subscriber %v subscribing to channel %v", i, s.channel)

					args := []string{"endpoint", "subscribe", "-c", s.channel, "-t", s.expire, "-u", ts.URL}
					output, err := execute(t, newRootCmd(), args...)
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
					args := []string{"endpoint", "publish", "-c", p.channel, "-m", p.message, "-u", ts.URL}
					<-time.After(p.wait)
					_, err := execute(t, newRootCmd(), args...)
					if err != nil {
						t.Errorf("Error executing publish: %v", err)
					}
				}()
			}

			wg.Wait()
		})
	}
}

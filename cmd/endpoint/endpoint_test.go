package endpoint

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
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
	out, err := execute(t, NewEndpointsCmd(), args...)

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
		case HTTPGetResponse:
			result = new(HTTPGetResponse)
		case HTTPPostResponse:
			result = new(HTTPPostResponse)
		case HTTPGetTTLResponse:
			result = new(HTTPGetTTLResponse)
		case StatusPlusErrorResponse:
			result = new(StatusPlusErrorResponse)
		}

		err = json.Unmarshal([]byte(out), &result)
		if err != nil {
			t.Error(err)
		}

		// Type switch to make the correct comparison with the reflect package
		switch expected := tt.response.(type) {
		case HTTPGetResponse:
			if !reflect.DeepEqual(result, &expected) {
				t.Errorf("got %v\nwant %v", result, &expected)
			}
		case HTTPPostResponse:
			if !reflect.DeepEqual(result, &expected) {
				t.Errorf("got %v\nwant %v", result, &expected)
			}
		case HTTPGetTTLResponse:
			if !reflect.DeepEqual(result, &expected) {
				t.Errorf("got %v\nwant %v", result, &expected)
			}
		case StatusPlusErrorResponse:
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
			response:     HTTPGetResponse{Status: 200, Key: "hello", Value: "world", Error: "null"},
			writeBadJSON: false,
			badURL:       false,
			shouldError:  false,
		},
		{
			name:             "Missing the key flag",
			alternateArgs:    []string{"get"},
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
			args := []string{"get", "-k", tt.key, "-u"}
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
			response:     StatusPlusErrorResponse{Status: 200, Error: "null"},
			writeBadJSON: false,
			badURL:       false,
			shouldError:  false,
		},
		{
			name:             "Missing the key flag",
			alternateArgs:    []string{"delete"},
			useAlternateArgs: true,
			shouldError:      true,
			expectedError:    "required",
		},
		badURLTest,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/v1/keys/{key}"
			args := []string{"delete", "-k", tt.key, "-u"}
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
			response:     StatusPlusErrorResponse{Status: 200, Error: "null"},
			writeBadJSON: false,
			badURL:       false,
			shouldError:  false,
		},
		{
			name:             "Missing the key flag",
			alternateArgs:    []string{"put", "-v", "world"},
			useAlternateArgs: true,
			shouldError:      true,
			expectedError:    "required",
		},
		{
			name:             "Missing the value flag",
			alternateArgs:    []string{"put", "-k", "hello"},
			useAlternateArgs: true,
			shouldError:      true,
			expectedError:    "required",
		},
		badURLTest,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/v1/keys/{key}"
			args := []string{"put", "-k", tt.key, "-v", tt.value, "-u"}
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
			response:     HTTPPostResponse{Status: 200, Key: "postKey", Error: "null"},
			writeBadJSON: false,
			badURL:       false,
			shouldError:  false,
		},
		{
			name:             "Missing the value flag",
			alternateArgs:    []string{"post"},
			useAlternateArgs: true,
			shouldError:      true,
			expectedError:    "required",
		},
		badURLTest,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/v1/keys"
			args := []string{"post", "-v", tt.value, "-u"}
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
			response:     HTTPGetTTLResponse{Status: 200, Key: "hello", TTL: intPtr(100), Error: "null"},
			writeBadJSON: false,
			badURL:       false,
			shouldError:  false,
		},
		{
			name:             "Missing the key flag",
			alternateArgs:    []string{"getTTL"},
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
			args := []string{"getTTL", "-k", tt.key, "-u"}
			if tt.useAlternateArgs {
				testHelper(t, tt, url, tt.alternateArgs)
			} else {
				testHelper(t, tt, url, args)
			}
		})
	}
}

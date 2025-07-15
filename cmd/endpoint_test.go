package cmd

import (
	"InMemoryDB/cmd/endpoint"
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
	name          string // Test case name
	key           string // Key for the request
	value         string // Value for the request
	returnStatus  int    // What the handler should set the status to
	response      any    // The response the handler should return
	writeBadJSON  bool   // Whether the server should write bad JSON
	badURL        bool   // Whether a bad url should be used or not
	shouldError   bool   // Whether the command should return an error
	expectedError string // A substring that a returned error should contain
}

// execute is a helper function for executing commands
func execute(t *testing.T, c *cobra.Command, args ...string) (string, error) {
	t.Helper()

	buf := new(bytes.Buffer)
	c.SetOut(buf)
	c.SetErr(buf)
	c.SetArgs(args)

	err := c.Execute()
	return strings.TrimSpace(buf.String()), err
}

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

func TestCommand_get(t *testing.T) {
	tests := []testCase{
		{
			name:         "Test no errors",
			key:          "hello",
			returnStatus: 200,
			response:     endpoint.HTTPGetResponse{Status: 200, Key: "hello", Value: "world", Error: ""},
			writeBadJSON: false,
			badURL:       false,
			shouldError:  false,
		},
		{
			name:          "Test bad JSON from server",
			key:           "hello",
			returnStatus:  200,
			writeBadJSON:  true,
			badURL:        false,
			shouldError:   true,
			expectedError: "error decoding response from server",
		},
		{
			name:          "Test bad JSON from server",
			key:           "hello",
			writeBadJSON:  false,
			badURL:        true,
			shouldError:   true,
			expectedError: "error creating request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Spin up mock server
			h := handlerHelper("/v1/keys/{key}", tt.returnStatus, tt.response, tt.writeBadJSON)
			ts := httptest.NewServer(h)
			defer ts.Close()

			args := []string{"endpoint", "get", "-k", tt.key, "-u"}
			if tt.badURL {
				args = append(args, "--bad-url")
			} else {
				args = append(args, ts.URL)
			}
			out, err := execute(t, rootCmd, args...)

			if (err != nil) != tt.shouldError {
				t.Errorf("expected shouldError(%v), got %v", tt.shouldError, err)
			}

			if tt.shouldError && !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("expected error to contain %v, got %v", tt.expectedError, err)
			}

			if !tt.shouldError {
				var result endpoint.HTTPGetResponse
				err = json.Unmarshal([]byte(out), &result)
				if err != nil {
					t.Error(err)
				}

				if !reflect.DeepEqual(tt.response, result) {
					t.Errorf("got %v\nwant %v", result, tt.response)
				}
			}
		})
	}
}

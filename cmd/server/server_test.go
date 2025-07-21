package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"reflect"
	"strings"
	"testing"
	"time"
)

// execute is a helper function for executing commands.
func execute(t *testing.T, c *cobra.Command, args ...string) (string, error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(1)*time.Second)
	defer cancel()
	c.SetContext(ctx)

	buf := new(bytes.Buffer)
	c.SetOut(buf)
	c.SetErr(buf)
	c.SetArgs(args)

	err := c.ExecuteContext(ctx)
	return strings.TrimSpace(buf.String()), err
}

func TestCommand_serve(t *testing.T) {
	testCases := []struct {
		name              string
		host              string
		startupFile       string
		shouldPersist     bool
		persistFile       string
		persistencePeriod int
	}{
		{
			name:              "Test configures database",
			host:              "localhost:8080",
			startupFile:       "testStartup.json",
			shouldPersist:     true,
			persistFile:       "persist.json",
			persistencePeriod: 30,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// go run main.go server serve -p 7070 -c 6 --persist --persist-file persist.json --startup-file startup.json
			// Execute command
			args := []string{"serve",
				"--startup-file", tt.startupFile,
				"-c", fmt.Sprintf("%v", tt.persistencePeriod),
				"--persist-file", tt.persistFile,
				"--Host", tt.host,
			}
			if tt.shouldPersist {
				args = append(args, "--persist")
			}
			out, err := execute(t, NewServerCmd(), args...)
			if err != nil {
				t.Error(err)
			}

			// Check Settings
			var result Settings
			err = json.Unmarshal([]byte(out), &result)
			expected := Settings{
				Host:              "localhost:8080",
				StartupFile:       tt.startupFile,
				ShouldPersist:     tt.shouldPersist,
				PersistFile:       tt.persistFile,
				PersistencePeriod: time.Duration(tt.persistencePeriod) * time.Second,
			}

			if !reflect.DeepEqual(result, expected) {
				t.Errorf("expected %v but got %v", expected, result)
			}
		})
	}
}

func TestCommand_serveValidation(t *testing.T) {
	t.Run("Test serve validation", func(t *testing.T) {
		_, err := execute(t, NewServerCmd(), []string{"serve", "--persist-file", "persist.json"}...)
		if err == nil {
			t.Error("Expected err but got nil")
		} else if !strings.Contains(err.Error(), "missing") {
			t.Errorf("Expected error to contain %v, got %v", "missing", err)
		}

		_, err = execute(t, NewServerCmd(), []string{"serve", "--persist"}...)
		if err == nil {
			t.Error("Expected err but got nil")
		} else if !strings.Contains(err.Error(), "missing") {
			t.Errorf("Expected error to contain %v, got %v", "missing", err)
		}
	})
}

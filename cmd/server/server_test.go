package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
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
		name                 string
		host                 string
		aofStartupFile       string
		shouldAofPersist     bool
		aofPersistFile       string
		aofPersistencePeriod int
		dbStartupFile        string
		shouldDbPersist      bool
		dbPersistFile        string
		dbPersistencePeriod  int
	}{
		{
			name:                 "With database startup file",
			host:                 "localhost:8080",
			shouldAofPersist:     true,
			aofPersistFile:       "aofPersistFile",
			aofPersistencePeriod: 10,
			dbStartupFile:        "testStartup.json",
			shouldDbPersist:      true,
			dbPersistFile:        "persist.json",
			dbPersistencePeriod:  30,
		},
		{
			name:                 "With aof startup file",
			host:                 "localhost:8080",
			aofStartupFile:       "aofStartup",
			shouldAofPersist:     true,
			aofPersistFile:       "aofPersistFile",
			aofPersistencePeriod: 10,
			shouldDbPersist:      true,
			dbPersistFile:        "persist.json",
			dbPersistencePeriod:  30,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// Execute command
			args := []string{"serve",
				"--aof-persist-cycle", fmt.Sprintf("%v", tt.aofPersistencePeriod),
				"--aof-persist-file", tt.aofPersistFile,
				"--db-persist-cycle", fmt.Sprintf("%v", tt.dbPersistencePeriod),
				"--db-persist-file", tt.dbPersistFile,
				"--host", tt.host,
			}
			fp := t.TempDir()
			if tt.aofStartupFile != "" {
				tt.aofStartupFile = filepath.Join(fp, tt.aofStartupFile)
				file, err := os.Create(tt.aofStartupFile)
				if err != nil {
					t.Fatal(err)
				}
				defer file.Close()
				args = append(args, "--aof-startup-file", tt.aofStartupFile)
			}
			if tt.dbStartupFile != "" {
				args = append(args, "--db-startup-file", tt.dbStartupFile)
			}
			if tt.shouldAofPersist {
				args = append(args, "--aof-persist")
			}
			if tt.shouldDbPersist {
				args = append(args, "--db-persist")
			}

			out, err := execute(t, NewServerCmd(), args...)
			if err != nil {
				t.Error(err)
			}

			// Scan the output for the JSON settings
			var jsonLines []string
			scanner := bufio.NewScanner(strings.NewReader(out))
			insideSettings := false
			for scanner.Scan() {
				line := scanner.Text()
				switch {
				case strings.Contains(line, "START_JSON_SETTINGS"):
					insideSettings = true
				case strings.Contains(line, "END_JSON_SETTINGS"):
					insideSettings = false
				default:
					if insideSettings {
						jsonLines = append(jsonLines, line)
					}
				}
			}
			actualJson := strings.Join(jsonLines, "\n")
			var result Settings
			err = json.Unmarshal([]byte(actualJson), &result)

			expected := Settings{
				Host:                      "localhost:8080",
				AofStartupFile:            tt.aofStartupFile,
				ShouldAofPersist:          tt.shouldAofPersist,
				AofPersistFile:            tt.aofPersistFile,
				AofPersistencePeriod:      time.Duration(tt.aofPersistencePeriod) * time.Second,
				DbStartupFile:             tt.dbStartupFile,
				ShouldDatabasePersist:     tt.shouldDbPersist,
				DatabasePersistFile:       tt.dbPersistFile,
				DatabasePersistencePeriod: time.Duration(tt.dbPersistencePeriod) * time.Second,
			}

			if !reflect.DeepEqual(result, expected) {
				t.Errorf("expected %v but got %v", expected, result)
			}
		})
	}
}

func TestCommand_serveValidation(t *testing.T) {
	t.Run("Test serve validation", func(t *testing.T) {
		// Should error if a db persistence file is specified but the database is not set to persist
		_, err := execute(t, NewServerCmd(), []string{"serve", "--db-persist-file", "persist.json"}...)
		if err == nil {
			t.Error("Expected err but got nil")
		} else if !strings.Contains(err.Error(), "missing") {
			t.Errorf("Expected error to contain %v, got %v", "missing", err)
		}

		// Should error if persistence is set to true but no file is provided
		_, err = execute(t, NewServerCmd(), []string{"serve", "--db-persist"}...)
		if err == nil {
			t.Error("Expected err but got nil")
		} else if !strings.Contains(err.Error(), "missing") {
			t.Errorf("Expected error to contain %v, got %v", "missing", err)
		}

		// Should error if an aof persistence file is specified but the aof is not set to persist
		_, err = execute(t, NewServerCmd(), []string{"serve", "--aof-persist-file", "aof"}...)
		if err == nil {
			t.Error("Expected err but got nil")
		} else if !strings.Contains(err.Error(), "missing") {
			t.Errorf("Expected error to contain %v, got %v", "missing", err)
		}

		// Should error if aof persistence is set to true but no file is provided
		_, err = execute(t, NewServerCmd(), []string{"serve", "--aof-persist"}...)
		if err == nil {
			t.Error("Expected err but got nil")
		} else if !strings.Contains(err.Error(), "missing") {
			t.Errorf("Expected error to contain %v, got %v", "missing", err)
		}

		// Should error if both an aof startup file and a database startup file are provided
		_, err = execute(t, NewServerCmd(), []string{"serve", "--aof-startup-file", "aof", "--db-startup-file", "db.json"}...)
		if err == nil {
			t.Error("Expected err but got nil")
		} else if !strings.Contains(err.Error(), "none of the others can be") {
			t.Errorf("Expected error to contain %v, got %v", "missing", err)
		}
	})
}

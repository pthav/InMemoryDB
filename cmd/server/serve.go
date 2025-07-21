package server

import (
	"InMemoryDB/database"
	"InMemoryDB/handler"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// Settings define user-configurable Settings for the database and http server
type Settings struct {
	Host              string        `json:"host"`              // The router's Host
	StartupFile       string        `json:"startupFile"`       // The startup file
	ShouldPersist     bool          `json:"shouldPersist"`     // Whether there should be persistence or not
	PersistFile       string        `json:"persistFile"`       // The file name for which to output persistence to
	PersistencePeriod time.Duration `json:"persistencePeriod"` // How long in between database persistence cycles
}

func newServeCmd() *cobra.Command {
	var host string
	var startupFile string
	var persistencePeriod int
	var persistFile string
	var shouldPersist bool

	// serveCmd serves up a database
	var serveCmd = &cobra.Command{
		Use:   "serve",
		Short: "Serve the database",
		Long: `Serve will spin up an in memory database instance and listen for localhost requests on the given port.
Flags can be provided to configure the database`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

			// Use args to create configuration functions
			var config []database.Options
			config = append(config, database.WithLogger(logger))
			config = append(config, database.WithPersistencePeriod(time.Duration(persistencePeriod)*time.Second))
			if shouldPersist {
				config = append(config, database.WithPersistence())
				config = append(config, database.WithPersistenceOutput(persistFile))
			}
			if startupFile != "" {
				config = append(config, database.WithInitialData(startupFile))
			}

			db, err := database.NewInMemoryDatabase(config...) // Configure database
			if err != nil {
				return err
			}

			dbSettings := db.GetSettings()
			s := Settings{
				Host:              host,
				StartupFile:       dbSettings.StartupFile,
				ShouldPersist:     dbSettings.ShouldPersist,
				PersistFile:       dbSettings.PersistFile,
				PersistencePeriod: dbSettings.PersistencePeriod,
			}
			out, err := json.MarshalIndent(s, "", "\t")
			if err != nil {
				return errors.New(fmt.Sprintf("error marshalling response: %v", err))
			}

			_, err = cmd.OutOrStdout().Write(append(out, '\n'))
			if err != nil {
				return err
			}

			h := &http.Server{Addr: host, Handler: handler.NewHandler(db, logger)}
			ctx := cmd.Context()
			go func() { // Allow server shutdown with a set context
				<-ctx.Done()
				log.Println("Shutting down server...")
				_ = h.Shutdown(context.Background())
			}()
			_ = h.ListenAndServe()
			return nil
		},
	}

	serveCmd.Flags().StringVarP(&host, "Host", "", "localhost:8080", "Host to listen for requests on")
	serveCmd.Flags().StringVar(&startupFile, "startup-file", "", "File containing json data to initialize the database with.")
	serveCmd.Flags().IntVarP(&persistencePeriod, "persist-cycle", "c", 60, "How long the persistence cycle should be in seconds.")
	serveCmd.Flags().StringVar(&persistFile, "persist-file", "", "File to persist the database to.")
	serveCmd.Flags().BoolVar(&shouldPersist, "persist", false, "Enables persistence.")
	serveCmd.MarkFlagsRequiredTogether("persist-file", "persist")

	return serveCmd
}

// go run main.go server serve -p 7070 -c 6 --persist --persist-file persist.json --startup-file startup.json
func init() {
}

package server

import (
	"InMemoryDB/database"
	"InMemoryDB/handler"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// Arguments
var startupFile string
var persistencePeriod int
var persistFile string
var shouldPersist bool

// serveCmd represents the serve command
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

		fmt.Printf("Created InMemoryDatabase instance running at localhost:%v\n", port)
		h := handler.NewHandler(db, logger)
		err = http.ListenAndServe(fmt.Sprintf("localhost:%v", port), h)
		if err != nil {
			return err
		}

		return nil
	},
}

// go run main.go serve -p 7070 -c 6 --persist --persist-file persist.json --startup-file startup.json
func init() {
	serveCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to listen on.")
	serveCmd.Flags().StringVar(&startupFile, "startup-file", "", "File containing json data to initialize the database with.")
	serveCmd.Flags().IntVarP(&persistencePeriod, "persist-cycle", "c", 60, "How long the persistence cycle should be in seconds.")
	serveCmd.Flags().StringVar(&persistFile, "persist-file", "", "File to persist the database to.")
	serveCmd.Flags().BoolVar(&shouldPersist, "persist", false, "Enables persistence.")
	serveCmd.MarkFlagsRequiredTogether("persist-file", "persist")
}

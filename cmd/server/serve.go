package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pthav/InMemoryDB/database"
	"github.com/pthav/InMemoryDB/handler"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

// Settings define user-configurable Settings for the database and http server
type Settings struct {
	Host                      string        `json:"host"`                      // The router's Host
	AofStartupFile            string        `json:"aofStartupFile"`            // The aof startup file
	ShouldAofPersist          bool          `json:"shouldAofPersist"`          // Whether there should be aof persistence or not
	AofPersistFile            string        `json:"aofPersistFile"`            // The file to output aof persistence to
	AofPersistencePeriod      time.Duration `json:"aofPersistencePeriod"`      // How long in between the aof persistence cycles
	DbStartupFile             string        `json:"dbStartupFile"`             // The database startup file
	ShouldDatabasePersist     bool          `json:"shouldDatabasePersist"`     // Whether there should be database persistence or not
	DatabasePersistFile       string        `json:"databasePersistFile"`       // The file name for which to output database persistence to
	DatabasePersistencePeriod time.Duration `json:"databasePersistencePeriod"` // How long in between database persistence cycles
}

// shutdown is called when the http server is shutting down gracefully
func shutdown(db *database.InMemoryDatabase, c *cobra.Command) {
	minWait := int64(1) // The minimum time to wait in seconds. This is exceeded only if shutdown functions take longer.
	_, _ = c.OutOrStdout().Write([]byte("Shutting down server...\n"))

	start := time.Now().Unix()
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		db.Shutdown()
	}()
	wg.Wait()

	// Only wait if minWait has not elapsed
	timeLeft := time.Duration(max(minWait-(time.Now().Unix()-start), int64(0))) * time.Second
	<-time.After(timeLeft)
}

func newServeCmd() *cobra.Command {
	var host string
	var aofStartupFile string
	var shouldAofPersist bool
	var aofPersistFile string
	var aofPersistencePeriod int
	var databaseStartupFile string
	var shouldDatabasePersist bool
	var databasePersistFile string
	var databasePersistencePeriod int
	var noLog bool

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
			if noLog {
				logger = slog.New(slog.NewTextHandler(io.Discard, nil))
			}
			config = append(config, database.WithLogger(logger))

			config = append(config, database.WithDatabasePersistencePeriod(time.Duration(databasePersistencePeriod)*time.Second))
			if shouldDatabasePersist {
				config = append(config, database.WithDatabasePersistence())
				config = append(config, database.WithDatabasePersistenceFile(databasePersistFile))
			}
			if databaseStartupFile != "" {
				config = append(config, database.WithInitialData(databaseStartupFile, true))
			}

			config = append(config, database.WithAofPersistencePeriod(time.Duration(aofPersistencePeriod)*time.Second))
			if shouldAofPersist {
				config = append(config, database.WithAofPersistenceFile(aofPersistFile))
				config = append(config, database.WithDatabasePersistenceFile(databasePersistFile))
			}
			if aofStartupFile != "" {
				config = append(config, database.WithInitialData(aofStartupFile, false))
			}

			db, err := database.NewInMemoryDatabase(config...) // Configure database
			if err != nil {
				return err
			}

			dbSettings := db.GetSettings()
			s := Settings{
				Host:                      host,
				AofStartupFile:            dbSettings.AofStartupFile,
				ShouldAofPersist:          shouldAofPersist,
				AofPersistFile:            dbSettings.AofPersistFile,
				AofPersistencePeriod:      dbSettings.AofPersistencePeriod,
				DbStartupFile:             dbSettings.DatabaseStartupFile,
				ShouldDatabasePersist:     dbSettings.ShouldDatabasePersist,
				DatabasePersistFile:       dbSettings.DatabasePersistFile,
				DatabasePersistencePeriod: dbSettings.DatabasePersistencePeriod,
			}
			out, err := json.MarshalIndent(s, "", "\t")
			if err != nil {
				return errors.New(fmt.Sprintf("error marshalling response: %v", err))
			}

			out = []byte(fmt.Sprintf("STARTING DATABASE\nSTART_JSON_SETTINGS\n%s\nEND_JSON_SETTINGS\n", string(out)))
			_, err = cmd.OutOrStdout().Write(out)
			if err != nil {
				return err
			}

			// This context will cancel either when the request is canceled or on shut down
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			h := &http.Server{
				Addr:    host,
				Handler: handler.NewHandler(db, logger),
				BaseContext: func(listener net.Listener) context.Context {
					return ctx
				},
			}

			shutdownWG := &sync.WaitGroup{} // Force server shutdown to wait
			shutdownWG.Add(1)
			h.RegisterOnShutdown(func() {
				go func() {
					defer shutdownWG.Done()
					shutdown(db, cmd)
				}()
			})

			g, gCtx := errgroup.WithContext(ctx)
			g.Go(func() error {
				return h.ListenAndServe()
			})
			g.Go(func() error { // Allow server shutdown with a set context
				<-gCtx.Done()
				err = h.Shutdown(context.Background())
				shutdownWG.Wait()
				return err
			})

			if err = g.Wait(); err != nil {
				_, _ = cmd.OutOrStdout().Write([]byte(fmt.Sprintf("exit reason: %v\n", err)))
			}
			return nil
		},
	}

	serveCmd.Flags().StringVarP(&host, "host", "", "localhost:8080", "Host to listen for requests on")
	serveCmd.Flags().BoolVar(&noLog, "no-log", false, "Disables logging output.")

	serveCmd.Flags().StringVar(&databaseStartupFile, "db-startup-file", "", "File containing json data to initialize the database with.")
	serveCmd.Flags().BoolVar(&shouldDatabasePersist, "db-persist", false, "Enables database persistence.")
	serveCmd.Flags().StringVar(&databasePersistFile, "db-persist-file", "", "File to persist the database to.")
	serveCmd.Flags().IntVarP(&databasePersistencePeriod, "db-persist-cycle", "", 60, "How long the database persistence cycle should be in seconds.")
	serveCmd.MarkFlagsRequiredTogether("db-persist-file", "db-persist")

	serveCmd.Flags().StringVar(&aofStartupFile, "aof-startup-file", "", "File containing aof data to initialize the database with.")
	serveCmd.Flags().BoolVar(&shouldAofPersist, "aof-persist", false, "Enables aof persistence.")
	serveCmd.Flags().StringVar(&aofPersistFile, "aof-persist-file", "", "File to persist aof data to.")
	serveCmd.Flags().IntVarP(&aofPersistencePeriod, "aof-persist-cycle", "", 1, "How long the aof persistence cycle should be in seconds.")
	serveCmd.MarkFlagsRequiredTogether("aof-persist-file", "aof-persist")

	serveCmd.MarkFlagsMutuallyExclusive("db-startup-file", "aof-startup-file")

	return serveCmd
}

func init() {
}

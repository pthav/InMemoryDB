package server

import (
	"InMemoryDB/database"
	"InMemoryDB/handler"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	Host              string        `json:"host"`              // The router's Host
	StartupFile       string        `json:"startupFile"`       // The startup file
	ShouldPersist     bool          `json:"shouldPersist"`     // Whether there should be persistence or not
	PersistFile       string        `json:"persistFile"`       // The file name for which to output persistence to
	PersistencePeriod time.Duration `json:"persistencePeriod"` // How long in between database persistence cycles
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
	var startupFile string
	var persistencePeriod int
	var persistFile string
	var shouldPersist bool
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
				config = append(config, database.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))))
			} else {
				config = append(config, database.WithLogger(logger))
			}
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

	serveCmd.Flags().StringVarP(&host, "Host", "", "localhost:8080", "Host to listen for requests on")
	serveCmd.Flags().StringVar(&startupFile, "startup-file", "", "File containing json data to initialize the database with.")
	serveCmd.Flags().IntVarP(&persistencePeriod, "persist-cycle", "c", 60, "How long the persistence cycle should be in seconds.")
	serveCmd.Flags().StringVar(&persistFile, "persist-file", "", "File to persist the database to.")
	serveCmd.Flags().BoolVar(&shouldPersist, "persist", false, "Enables persistence.")
	serveCmd.Flags().BoolVar(&noLog, "no-log", false, "Disables logging output.")
	serveCmd.MarkFlagsRequiredTogether("persist-file", "persist")

	return serveCmd
}

func init() {
}

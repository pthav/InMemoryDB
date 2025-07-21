package database

import (
	"encoding/json"
	"log/slog"
	"os"
	"time"
)

// settings define user-configurable settings for the database in a single struct
type settings struct {
	startupFile       string        // The startup file
	shouldPersist     bool          // Whether there should be persistence or not
	persistFile       string        // The file name for which to output persistence to
	persistencePeriod time.Duration // How long in between database persistence cycles
	logger            *slog.Logger  // Logging
}

type Options func(*InMemoryDatabase) error

// WithPersistence sets the database to persist
func WithPersistence() Options {
	return func(db *InMemoryDatabase) error {
		db.s.shouldPersist = true
		return nil
	}
}

// WithPersistenceOutput sets the filename
func WithPersistenceOutput(s string) Options {
	return func(db *InMemoryDatabase) error {
		db.s.persistFile = s
		return nil
	}
}

// WithPersistencePeriod sets the persistence period
func WithPersistencePeriod(d time.Duration) Options {
	return func(db *InMemoryDatabase) error {
		db.s.persistencePeriod = d
		return nil
	}
}

// WithLogger sets the logger to be used
func WithLogger(l *slog.Logger) Options {
	return func(db *InMemoryDatabase) error {
		db.s.logger = l
		return nil
	}
}

// WithInitialData allows the provision of a .json file to initialize the database with
func WithInitialData(filename string) Options {
	return func(db *InMemoryDatabase) error {
		db.s.startupFile = filename
		data, err := os.ReadFile(filename)
		if err != nil {
			return err
		}

		err = json.Unmarshal(data, db)
		if err != nil {
			return err
		}
		return nil
	}
}

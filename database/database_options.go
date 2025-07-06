package database

import (
	"encoding/json"
	"log/slog"
	"os"
	"time"
)

// settings define user-configurable settings for the database in a single struct
type settings struct {
	shouldPersist     bool          // Whether there should be persistence or not
	persistFile       string        // The file name for which to output persistence to
	persistencePeriod time.Duration // How long in between database persistence cycles
	logger            *slog.Logger  // Logging
}

type Options func(*InMemoryDatabase)

// WithPersistenceOutput sets the filename
func WithPersistenceOutput(s string) Options {
	return func(db *InMemoryDatabase) {
		db.s.persistFile = s
	}
}

// WithPersistencePeriod sets the persistence period
func WithPersistencePeriod(d time.Duration) Options {
	return func(db *InMemoryDatabase) {
		db.s.persistencePeriod = d
	}
}

// WithLogger sets the logger to be used
func WithLogger(l *slog.Logger) Options {
	return func(db *InMemoryDatabase) {
		db.s.logger = l
	}
}

// WithInitialData allows the provision of a .json file to initialize the database with
func WithInitialData(filename string) Options {
	return func(db *InMemoryDatabase) {
		data, err := os.ReadFile(filename)
		if err != nil {
			panic("Failed to open file passed to WithInitialData during InMemoryDatabase initialization")
		}

		err = json.Unmarshal(data, db)
		if err != nil {
			panic("Failed to unmarshal into new database during InMemoryDatabase initialization")
		}
	}
}

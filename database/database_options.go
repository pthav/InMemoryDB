package database

import (
	"bufio"
	"encoding/json"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

// settings define user-configurable settings for the database in a single struct
type settings struct {
	aofStartupFile            string        // The aof startup file
	shouldAofPersist          bool          // Whether there should be AOF persistence or not
	aofPersistenceFile        string        // The file name for which to output AOF persistence to
	aofPersistencePeriod      time.Duration // How long in between AOF persistence cycles
	databaseStartupFile       string        // The database startup file
	shouldDatabasePersist     bool          // Whether there should be database persistence or not
	databasePersistenceFile   string        // The file name for which to output database persistence to
	databasePersistencePeriod time.Duration // How long in between database persistence cycles
	logger                    *slog.Logger  // Logging
}

type Options func(*InMemoryDatabase) error

// WithAofPersistence enables AOF persistence
func WithAofPersistence() Options {
	return func(db *InMemoryDatabase) error {
		db.s.shouldAofPersist = true
		return nil
	}
}

// WithAofPersistenceFile sets the file name to persist the AOF to
func WithAofPersistenceFile(s string) Options {
	return func(db *InMemoryDatabase) error {
		db.s.aofPersistenceFile = s
		return nil
	}
}

// WithAofPersistencePeriod sets the period between AOF persistence cycles
func WithAofPersistencePeriod(d time.Duration) Options {
	return func(db *InMemoryDatabase) error {
		db.s.aofPersistencePeriod = d
		return nil
	}
}

// WithDatabasePersistence enables database persistence
func WithDatabasePersistence() Options {
	return func(db *InMemoryDatabase) error {
		db.s.shouldDatabasePersist = true
		return nil
	}
}

// WithDatabasePersistenceFile sets the filename to persist the database to
func WithDatabasePersistenceFile(s string) Options {
	return func(db *InMemoryDatabase) error {
		db.s.databasePersistenceFile = s
		return nil
	}
}

// WithDatabasePersistencePeriod sets the period between database persistence cycles
func WithDatabasePersistencePeriod(d time.Duration) Options {
	return func(db *InMemoryDatabase) error {
		db.s.databasePersistencePeriod = d
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

// WithInitialData allows the provision of a .json file to initialize the database with. When persistenceType is true,
// the file is specified to be a database persistence file. When it is false, the file is specified to be an AOF file.
func WithInitialData(filename string, persistenceType bool) Options {
	return func(db *InMemoryDatabase) error {
		if persistenceType {
			db.s.databaseStartupFile = filename
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

		db.s.aofStartupFile = filename
		file, err := os.Open(filename)
		if err != nil {
			return err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			args := strings.Split(line, " ")
			switch args[0] {
			case "PUT":
				if len(args) != 4 {
					continue
				}
				key := args[1]

				d := databaseEntry{
					value: args[2],
					ttl:   nil,
				}

				if args[3] != "-1" {
					ttlInt, err := strconv.Atoi(args[3])
					if err != nil {
						continue
					}
					var ttl int64
					ttl = int64(ttlInt)
					d.ttl = &ttl
				}

				db.store(key, d)
			case "DELETE":
				if len(args) != 2 {
					continue
				}

				db.delete(args[1])
			}
		}

		return nil
	}
}

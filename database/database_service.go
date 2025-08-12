package database

import (
	"container/heap"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"log/slog"
	"os"
	"sync"
	"time"
)

type databaseEntry struct {
	value string
	ttl   *int64
}

type dbStore map[string]databaseEntry

// InMemoryDatabase stores data in memory using a sync map to ensure thread safety. Receiver methods for
// InMemoryDatabase assume already validated inputs. For example, in Put, the key and value should not be empty.
type InMemoryDatabase struct {
	database dbStore       // Store the database key, value pairs
	ttl      *ttlHeap      // Store TTLs on a heap
	mu       sync.RWMutex  // Mutex for coordinating ttlHeap cleaner and other operations
	newItem  chan struct{} // This channel tells the cleaner routine when a ttl has been created/updated
	s        settings      // Database settings
}

// NewInMemoryDatabase returns a new InMemoryDatabase instance
func NewInMemoryDatabase(opts ...Options) (db *InMemoryDatabase, err error) {
	db = &InMemoryDatabase{
		database: dbStore{},
		ttl:      &ttlHeap{},
		mu:       sync.RWMutex{},
		newItem:  make(chan struct{}, 1),
		s: settings{
			shouldAofPersist:          false,
			aofPersistenceFile:        "persistAof",
			aofPersistencePeriod:      time.Second,
			shouldDatabasePersist:     false,
			databasePersistenceFile:   "persistDatabase.json",
			databasePersistencePeriod: 5 * time.Minute,
			logger:                    slog.New(slog.NewTextHandler(os.Stdout, nil)),
		},
	}
	heap.Init(db.ttl)

	for _, c := range opts {
		err = c(db)
		if err != nil {
			return
		}
	}

	go db.ttlCleanup()
	if db.s.shouldAofPersist {
		go db.persistAofCycle()
	}

	if db.s.shouldDatabasePersist {
		go db.persistDatabaseCycle()
	}

	return
}

// Shutdown will persistDatabase one last time if it is enabled.
func (i *InMemoryDatabase) Shutdown() {
	if i.s.shouldAofPersist {
		i.persistAof()
	}

	if i.s.shouldDatabasePersist {
		i.persistDatabase()
	}
}

// GetSettings returns the database settings so that the settings struct does not have to be an exported type
func (i *InMemoryDatabase) GetSettings() struct {
	AofStartupFile            string
	ShouldAofPersist          bool
	AofPersistFile            string
	AofPersistencePeriod      time.Duration
	DatabaseStartupFile       string
	ShouldDatabasePersist     bool
	DatabasePersistFile       string
	DatabasePersistencePeriod time.Duration
} {
	return struct {
		AofStartupFile            string
		ShouldAofPersist          bool
		AofPersistFile            string
		AofPersistencePeriod      time.Duration
		DatabaseStartupFile       string
		ShouldDatabasePersist     bool
		DatabasePersistFile       string
		DatabasePersistencePeriod time.Duration
	}{
		AofStartupFile:            i.s.aofStartupFile,
		ShouldAofPersist:          i.s.shouldAofPersist,
		AofPersistFile:            i.s.aofPersistenceFile,
		AofPersistencePeriod:      i.s.aofPersistencePeriod,
		DatabaseStartupFile:       i.s.databaseStartupFile,
		ShouldDatabasePersist:     i.s.shouldDatabasePersist,
		DatabasePersistFile:       i.s.databasePersistenceFile,
		DatabasePersistencePeriod: i.s.databasePersistencePeriod,
	}
}

// Create a key value pair in the database
func (i *InMemoryDatabase) Create(data struct {
	Value string `json:"value"`
	Ttl   *int64 `json:"ttl"`
}) (bool, string) {
	i.mu.Lock()
	defer i.mu.Unlock()

	id := uuid.New().String()
	newEntry := databaseEntry{value: data.Value}
	var ttl int64
	if data.Ttl != nil {
		ttl = *data.Ttl + time.Now().Unix()
		newEntry.ttl = &ttl
	}
	_, loaded := i.loadOrStore(id, newEntry)
	if data.Ttl != nil && !loaded {
		heap.Push(i.ttl, ttlHeapData{id, ttl})

		// Notify cleaner of new TTL
		select {
		case i.newItem <- struct{}{}:
		default:
		}
	}

	if data.Ttl != nil {
		i.appendToAof(fmt.Sprintf(`PUT %s %s %v`, id, data.Value, *data.Ttl))
	} else {
		i.appendToAof(fmt.Sprintf(`PUT %s %s %v`, id, data.Value, -1))
	}

	return !loaded, id
}

// Get a value from the database by key if it exists and is valid
func (i *InMemoryDatabase) Get(key string) (string, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	dbEntry, loaded := i.load(key)
	if (loaded && dbEntry.ttl == nil) || (loaded && *dbEntry.ttl > time.Now().Unix()) {
		return dbEntry.value, true
	}
	return "", false
}

// GetTTL the remaining TTL for a given key
func (i *InMemoryDatabase) GetTTL(key string) (*int64, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	dbEntry, loaded := i.load(key)
	if !loaded || (dbEntry.ttl != nil && *dbEntry.ttl <= time.Now().Unix()) {
		return nil, false
	} else if dbEntry.ttl != nil {
		var ttl int64
		ttl = *dbEntry.ttl - time.Now().Unix()
		return &ttl, true
	}
	return nil, true
}

// Put a key value pair into the database.
func (i *InMemoryDatabase) Put(data struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Ttl   *int64 `json:"ttl"`
}) bool {
	i.mu.Lock()
	defer i.mu.Unlock()

	if data.Ttl != nil {
		i.appendToAof(fmt.Sprintf(`PUT %s %s %v`, data.Key, data.Value, *data.Ttl))
	} else {
		i.appendToAof(fmt.Sprintf(`PUT %s %s %v`, data.Key, data.Value, -1))
	}

	_, loaded := i.load(data.Key)
	newEntry := databaseEntry{value: data.Value}
	var ttl int64
	if data.Ttl != nil {
		ttl = *data.Ttl + time.Now().Unix()
		newEntry.ttl = &ttl
	}
	i.store(data.Key, newEntry)

	if data.Ttl != nil {
		heap.Push(i.ttl, ttlHeapData{data.Key, ttl})

		// Notify cleaner of new TTL
		select {
		case i.newItem <- struct{}{}:
		default:
		}
	}

	return loaded
}

// Delete a key value pair from the database
func (i *InMemoryDatabase) Delete(key string) bool {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.appendToAof(fmt.Sprintf(`DELETE %s`, key))

	_, loaded := i.loadAndDelete(key)
	return loaded
}

// ttlCleanup performs routine ttlHeap cleanup
func (i *InMemoryDatabase) ttlCleanup() {
	i.s.logger.Info("starting ttl cleanup routine")
	for {
		i.mu.Lock()

		if len(*i.ttl) == 0 {
			i.mu.Unlock()
			<-i.newItem
			continue
		}

		// Get the earliest expiring ttl and a delay from now until it is expired
		next := i.ttl.Peak().(ttlHeapData).ttl
		now := time.Now().Unix()
		delay := next - now

		i.mu.Unlock()

		// Wait until either a new item is created or the delay has finished
		if delay > 0 {
			select {
			case <-time.After(time.Duration(delay) * time.Second):
			case <-i.newItem:
				i.s.logger.Info("ttl cleanup routine new item")
				continue
			}
		}

		i.mu.Lock()
		for len(*i.ttl) > 0 {
			timeLeft := i.ttl.Peak().(ttlHeapData).ttl - time.Now().Unix()
			if timeLeft > 0 {
				break
			}

			heapData := heap.Pop(i.ttl).(ttlHeapData)
			key := heapData.key
			ttl := heapData.ttl

			// Delete only if it still exists and the ttl has not been modified
			dbEntry, loaded := i.load(key)
			if loaded && dbEntry.ttl != nil && *dbEntry.ttl == ttl {
				i.appendToAof(fmt.Sprintf(`DELETE %s`, key))
				i.delete(key)
			}
		}
		i.mu.Unlock()
	}
}

// appendToAof will append a line to the AOF file. This function assumes a lock has been acquired.
func (i *InMemoryDatabase) appendToAof(line string) {
	if !i.s.shouldAofPersist {
		return
	}

	file, err := os.OpenFile(i.s.aofPersistenceFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		i.s.logger.Error("failed to open aof persistence file", "err", err)
		return
	}
	defer func() {
		err = file.Close()
		if err != nil {
			i.s.logger.Error("error closing persistence file: ", "err", err)
			return
		}
	}()

	_, err = file.WriteString(line + "\n")
	if err != nil {
		i.s.logger.Error("failed to append to aof persistence file", "err", err)
		return
	}
}

// persistAofCycle will call the persistAof function based on a configured period
func (i *InMemoryDatabase) persistAofCycle() {
	i.s.logger.Info("starting AOF persistence routine")
	for {
		<-time.After(time.Second)
		i.persistAof()
	}
}

// persistAof will sync the AOF file to make sure all changes are up to date
func (i *InMemoryDatabase) persistAof() {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.s.logger.Info("attempting to persist aof data")

	file, err := os.OpenFile(i.s.aofPersistenceFile, os.O_SYNC|os.O_CREATE, 0644)
	if err != nil {
		i.s.logger.Error("failed to open aof persistence file", "err", err)
		return
	}
	defer func() {
		err = file.Close()
		if err != nil {
			i.s.logger.Error("error closing persistence file: ", "err", err)
			return
		}
	}()

	err = file.Sync()
	if err != nil {
		i.s.logger.Error("failed to sync aof persistence file", "err", err)
		return
	}
}

// persistDatabaseCycle will call the persistDatabase function based on a configured period
func (i *InMemoryDatabase) persistDatabaseCycle() {
	i.s.logger.Info("starting database persistence routine")
	for {
		<-time.After(i.s.databasePersistencePeriod)
		i.persistDatabase()
	}
}

// persistDatabase will attempt to persistDatabase all storage data to the configured output file
func (i *InMemoryDatabase) persistDatabase() {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.s.logger.Info("attempting to persist database data")

	// Make sure the file is open
	file, err := os.Create(i.s.databasePersistenceFile)
	defer func() {
		err = file.Close()
		if err != nil {
			i.s.logger.Error("error closing persistence file: ", "err", err)
			return
		}
	}()

	if err != nil {
		i.s.logger.Error("error opening/creating persistence file: ", "err", err)
		return
	}

	data, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		i.s.logger.Error("error marshaling database: ", "err", err)
		return
	}

	_, err = file.Write(data)
	if err != nil {
		i.s.logger.Error("error writing database json to file: ", "err", err)
		return
	}
}

// These helper functions assume the caller has locked the database mutex

// If the key exists in the database, return the associated entry alongside True.
// Otherwise, return the zero value alongside False.
func (i *InMemoryDatabase) load(key string) (databaseEntry, bool) {
	d, loaded := i.database[key]
	return d, loaded
}

// Delete the key value pair from the database
func (i *InMemoryDatabase) delete(key string) {
	delete(i.database, key)
}

// If the key exists in the database, delete it and return the deleted entry alongside True.
// Otherwise, return a zero value alongside False.
func (i *InMemoryDatabase) loadAndDelete(key string) (databaseEntry, bool) {
	d, loaded := i.load(key)
	i.delete(key)
	return d, loaded
}

// Store the key value pair in the database
func (i *InMemoryDatabase) store(key string, d databaseEntry) {
	i.database[key] = d
}

// If the key exists in the database storage, loadOrStore will return the existing entry and True.
// Otherwise, it will return the new entry and False.
func (i *InMemoryDatabase) loadOrStore(key string, d databaseEntry) (databaseEntry, bool) {
	o, loaded := i.load(key)
	if loaded {
		return o, true
	}

	i.store(key, d)
	return d, false
}

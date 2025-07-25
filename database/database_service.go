package database

import (
	"container/heap"
	"encoding/json"
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
			shouldPersist:     false,
			persistFile:       "persist.json",
			persistencePeriod: 5 * time.Minute,
			logger:            slog.New(slog.NewTextHandler(os.Stdout, nil)),
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
	if db.s.shouldPersist {
		go db.persistCycle()
	}

	return
}

// Shutdown will persist one last time if it is enabled.
func (i *InMemoryDatabase) Shutdown() {
	if i.s.shouldPersist {
		i.persist()
	}
}

// GetSettings returns the database settings so that the settings struct does not have to be an exported type
func (i *InMemoryDatabase) GetSettings() struct {
	StartupFile       string
	ShouldPersist     bool
	PersistFile       string
	PersistencePeriod time.Duration
} {
	return struct {
		StartupFile       string
		ShouldPersist     bool
		PersistFile       string
		PersistencePeriod time.Duration
	}{
		StartupFile:       i.s.startupFile,
		ShouldPersist:     i.s.shouldPersist,
		PersistFile:       i.s.persistFile,
		PersistencePeriod: i.s.persistencePeriod,
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
				i.delete(key)
			}
		}
		i.mu.Unlock()
	}
}

// persistCycle will call the persist function based on a configured period
func (i *InMemoryDatabase) persistCycle() {
	i.s.logger.Info("starting persistence routine")
	for {
		<-time.After(i.s.persistencePeriod)
		i.persist()
	}
}

// persist will attempt to persist all storage data to the configured output file
func (i *InMemoryDatabase) persist() {
	i.mu.Lock()
	i.s.logger.Info("attempting to persist data")

	// Make sure the file is open
	file, err := os.Create(i.s.persistFile)

	if err != nil {
		i.s.logger.Error("Error opening/creating persistence file: ", "err", err)
		i.mu.Unlock()
		return
	}

	data, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		i.s.logger.Error("Error marshaling database: ", "err", err)
		i.mu.Unlock()
		return
	}

	_, err = file.Write(data)
	if err != nil {
		i.s.logger.Error("Error writing database json to file: ", "err", err)
		i.mu.Unlock()
		return
	}

	err = file.Close()
	if err != nil {
		i.s.logger.Error("Error closing persistence file: ", "err", err)
		i.mu.Unlock()
		return
	}
	i.mu.Unlock()
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

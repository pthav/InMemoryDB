package database

import (
	"container/heap"
	"github.com/google/uuid"
	"sync"
	"time"
)

type databaseEntry struct {
	value string
	ttl   *int64
}

// InMemoryDatabase stores data in memory using a sync map to ensure thread safety
type InMemoryDatabase struct {
	store   sync.Map   // Store the database key, value pairs
	ttl     *ttlHeap   // Store TTLs on a heap
	mu      sync.Mutex // Mutex for coordinating ttlHeap cleaner with other operations
	newItem chan struct{}
}

// NewInMemoryDatabase Return a new InMemoryDatabase instance
func NewInMemoryDatabase() *InMemoryDatabase {
	db := &InMemoryDatabase{}
	db.store = sync.Map{}
	db.ttl = new(ttlHeap)
	heap.Init(db.ttl)
	db.newItem = make(chan struct{})
	db.mu = sync.Mutex{}
	go db.TTLCleanup()
	return db
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
	_, loaded := i.store.LoadOrStore(id, newEntry)
	if data.Ttl != nil && !loaded {
		i.ttl.Push(ttlHeapData{id, ttl})

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
	value, loaded := i.store.Load(key)
	if (loaded && value.(databaseEntry).ttl == nil) || (loaded && *value.(databaseEntry).ttl > time.Now().Unix()) {
		return value.(databaseEntry).value, true
	}
	return "", false
}

// GetTTL the remaining TTL for a given key
func (i *InMemoryDatabase) GetTTL(key string) (int64, bool) {
	value, loaded := i.store.Load(key)
	if loaded && value.(databaseEntry).ttl != nil {
		return *value.(databaseEntry).ttl, true
	}
	return 0, false
}

// Put a key value pair into the database
func (i *InMemoryDatabase) Put(data struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Ttl   *int64 `json:"ttl"`
}) bool {
	i.mu.Lock()
	defer i.mu.Unlock()

	_, loaded := i.store.LoadOrStore(data.Key, data.Value)
	newEntry := databaseEntry{value: data.Value}
	var ttl int64
	if data.Ttl != nil {
		ttl = *data.Ttl + time.Now().Unix()
		newEntry.ttl = &ttl
	}
	i.store.Store(data.Key, newEntry)
	if data.Ttl != nil {
		i.ttl.Push(ttlHeapData{data.Key, ttl})

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

	_, loaded := i.store.LoadAndDelete(key)
	return loaded
}

// TTLCleanup performs routine ttlHeap cleanup
func (i *InMemoryDatabase) TTLCleanup() {
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
				continue
			}
		}

		i.mu.Lock()
		heapData := i.ttl.Pop().(ttlHeapData)
		key := heapData.key
		ttl := heapData.ttl

		// Delete only if it still exists and the ttl has not been modified
		data, loaded := i.store.Load(key)
		if loaded && data.(databaseEntry).ttl != nil && *data.(databaseEntry).ttl == ttl {
			i.store.Delete(key)
		}
		i.mu.Unlock()
	}

}

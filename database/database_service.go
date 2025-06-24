package database

import (
	"container/heap"
	"github.com/google/uuid"
	"sync"
	"time"
)

type databaseEntry struct {
	value string
	ttl   int64
}

// InMemoryDatabase stores data in memory using a sync map to ensure thread safety
type InMemoryDatabase struct {
	store sync.Map
	ttl   *ttlHeap
}

// NewInMemoryDatabase Return a new InMemoryDatabase instance
func NewInMemoryDatabase() *InMemoryDatabase {
	db := &InMemoryDatabase{}
	db.store = sync.Map{}
	db.ttl = new(ttlHeap)
	heap.Init(db.ttl)
	return db
}

// Create a key value pair in the database
func (i *InMemoryDatabase) Create(data struct {
	Value string `json:"value"`
	Ttl   *int64 `json:"ttl"`
}) (bool, string) {
	id := uuid.New().String()
	newEntry := databaseEntry{value: data.Value}
	var ttl int64
	if data.Ttl != nil {
		ttl = *data.Ttl + time.Now().Unix()
		newEntry.ttl = ttl
	}
	_, loaded := i.store.LoadOrStore(id, newEntry)
	if data.Ttl != nil && !loaded {
		i.ttl.Push(ttlHeapData{id, ttl})
	}
	return !loaded, id
}

// Get a value from the database by key
func (i *InMemoryDatabase) Get(key string) (string, bool) {
	value, loaded := i.store.Load(key)
	if loaded {
		return value.(databaseEntry).value, true
	}
	return "", false
}

func (i *InMemoryDatabase) GetTTL(key string) (int64, bool) {
	value, loaded := i.store.Load(key)
	if loaded {
		return value.(databaseEntry).ttl, true
	}
	return 0, false
}

// Put a key value pair into the database
func (i *InMemoryDatabase) Put(data struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Ttl   *int64 `json:"ttl"`
}) bool {
	_, loaded := i.store.LoadOrStore(data.Key, data.Value)
	newEntry := databaseEntry{value: data.Value}
	var ttl int64
	if data.Ttl != nil {
		ttl = *data.Ttl + time.Now().Unix()
		newEntry.ttl = ttl
	}
	i.store.Store(data.Key, databaseEntry{data.Value, ttl})
	if data.Ttl != nil {
		i.ttl.Push(ttlHeapData{data.Key, ttl})
	}
	return loaded
}

// Delete a key value pair from the database
func (i *InMemoryDatabase) Delete(key string) bool {
	_, loaded := i.store.LoadAndDelete(key)
	return loaded
}

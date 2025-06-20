package database

import (
	"container/heap"
	"github.com/google/uuid"
	"sync"
	"time"
)

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
	Ttl   *int   `json:"ttl"`
}) (bool, string) {
	id := uuid.New().String()
	_, loaded := i.store.LoadOrStore(id, data.Value)
	if data.Ttl != nil && !loaded {
		i.ttl.Push(keyTtl{id, int64(*data.Ttl) + time.Now().Unix()})
	}
	return !loaded, id
}

// Get a value from the database by key
func (i *InMemoryDatabase) Read(key string) (string, bool) {
	value, loaded := i.store.Load(key)
	if loaded {
		return value.(string), true
	}
	return "", false
}

// Update a key value pair in the database if it exists. Otherwise, Create a key value pair.
func (i *InMemoryDatabase) Update(data struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Ttl   *int   `json:"ttl"`
}) bool {
	_, loaded := i.store.LoadOrStore(data.Key, data.Value)
	i.store.Store(data.Key, data.Value)
	if data.Ttl != nil {
		i.ttl.Push(keyTtl{data.Key, int64(*data.Ttl) + time.Now().Unix()})
	}
	return loaded
}

// Delete a key value pair from the database
func (i *InMemoryDatabase) Delete(key string) bool {
	_, loaded := i.store.LoadAndDelete(key)
	return loaded
}

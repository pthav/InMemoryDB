package database

import (
	"sync"
)

// InMemoryDatabase stores data in memory using a sync map to ensure thread safety
type InMemoryDatabase struct {
	store sync.Map
}

// NewInMemoryDatabase Return a new InMemoryDatabase instance
func NewInMemoryDatabase() *InMemoryDatabase {
	db := &InMemoryDatabase{}
	db.store = sync.Map{}
	return db
}

// Create a key value pair in the database
func (i *InMemoryDatabase) Create(key string, value string) bool {
	_, loaded := i.store.LoadOrStore(key, value)
	return !loaded
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
func (i *InMemoryDatabase) Update(key string, value string) bool {
	_, loaded := i.store.LoadOrStore(key, value)
	i.store.Store(key, value)
	return loaded
}

// Delete a key value pair from the database
func (i *InMemoryDatabase) Delete(key string) bool {
	_, loaded := i.store.LoadAndDelete(key)
	return loaded
}

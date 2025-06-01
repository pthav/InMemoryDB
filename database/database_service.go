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

// Set a key value pair in the database
func (i *InMemoryDatabase) Set(key string, value string) bool {
	i.store.Store(key, value)
	return true
}

// Get a value from the database by key
func (i *InMemoryDatabase) Get(key string) (string, bool) {
	loaded, ok := i.store.Load(key)
	if ok {
		return loaded.(string), true
	}
	return "", false
}

// Delete a key value pair from the database
func (i *InMemoryDatabase) Delete(key string) bool {
	i.store.Delete(key)
	return true
}

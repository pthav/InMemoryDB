package tests

import (
	"InMemoryDB/database"
	"testing"
	"time"
)

// createStartupFile will create a startup file with randomly generated data.
//func createStartupFile(samples int) {
//	db, _ := database.NewInMemoryDatabase(
//		database.WithLogger(discardLogger),
//		database.WithPersistence(),
//		database.WithPersistencePeriod(time.Duration(1)*time.Second),
//	)
//
//	for i := 0; i < samples; i++ {
//		db.Create(generatePost())
//	}
//	db.Shutdown()
//}

//func TestT_make(t *testing.T) {
//	createStartupFile(1000)
//}

// FuzzDB fuzzes tests for InMemoryDB.Create
func FuzzDBCreate(f *testing.F) {
	f.Fuzz(func(t *testing.T, value string, useTTL bool, ttl int64, waitExpire bool) {
		if value == "" || ttl <= 0 || ttl > 5 || (!useTTL && waitExpire) {
			t.Skip("Invalid test case")
		}

		db, _ := database.NewInMemoryDatabase(
			database.WithLogger(discardLogger),
			database.WithInitialData("startup.json"),
		)

		createRequest := struct {
			Value string `json:"value"`
			Ttl   *int64 `json:"ttl"`
		}{
			Value: value,
		}

		if useTTL {
			createRequest.Ttl = &ttl
		}

		created, key := db.Create(createRequest)
		if !created {
			t.Skip("Hash collision")
		}

		if waitExpire {
			<-time.After(time.Duration(ttl) * time.Second)
		}

		// Value
		expectedExists := !waitExpire
		storedValue, exists := db.Get(key)
		if exists != expectedExists {
			t.Fatalf("Value found with boolean %v. Expected %v", exists, expectedExists)
		}

		if expectedExists && storedValue != value {
			t.Errorf("Expected value %v, but got %v", value, storedValue)
		}

		// TTL
		storedTTL, exists := db.GetTTL(key)
		if exists != expectedExists {
			t.Fatalf("TTL found with boolean %v. Expected %v", exists, expectedExists)
		}

		if expectedExists && useTTL && *storedTTL != ttl {
			t.Errorf("Expected TTL %v, but got %v", ttl, *storedTTL)
		} else if expectedExists && !useTTL && storedTTL != nil {
			t.Error("Expected TTL to be nil")
		}
	})
}

// FuzzDB fuzzes tests for InMemoryDB.Put
func FuzzDBPut(f *testing.F) {
	f.Fuzz(func(t *testing.T, key string, value string, useTTL bool, ttl int64, waitExpire bool) {
		if key == "" || value == "" || ttl <= 0 || ttl > 5 || (!useTTL && waitExpire) {
			t.Skip("Invalid test case")
		}

		db, _ := database.NewInMemoryDatabase(
			database.WithLogger(discardLogger),
			database.WithInitialData("startup.json"),
		)

		_, exists := db.Get(key)

		putRequest := struct {
			Key   string `json:"key"`
			Value string `json:"value"`
			Ttl   *int64 `json:"ttl"`
		}{
			Key:   key,
			Value: value,
			Ttl:   nil,
		}

		if useTTL {
			putRequest.Ttl = &ttl
		}

		updated := db.Put(putRequest)
		if updated != exists {
			t.Errorf("Mismatch between exists and update, %v and %v", exists, updated)
		}

		if waitExpire {
			<-time.After(time.Duration(ttl) * time.Second)
		}

		// Value
		expectedExists := !waitExpire
		storedValue, exists := db.Get(key)
		if exists != expectedExists {
			t.Fatalf("Value found with boolean %v. Expected %v", exists, expectedExists)
		}

		if expectedExists && storedValue != value {
			t.Errorf("Expected value %v, but got %v", value, storedValue)
		}

		// TTL
		storedTTL, exists := db.GetTTL(key)
		if exists != expectedExists {
			t.Fatalf("TTL found with boolean %v. Expected %v", exists, expectedExists)
		}

		if expectedExists && useTTL && *storedTTL != ttl {
			t.Errorf("Expected TTL %v, but got %v", ttl, *storedTTL)
		} else if expectedExists && !useTTL && storedTTL != nil {
			t.Error("Expected TTL to be nil")
		}
	})
}

// FuzzDBDelete fuzzes tests for InMemoryDB.Delete
func FuzzDBDelete(f *testing.F) {
	f.Fuzz(func(t *testing.T, key string) {
		db, _ := database.NewInMemoryDatabase(
			database.WithLogger(discardLogger),
			database.WithInitialData("startup.json"),
		)

		_, exists := db.Get(key)
		deleted := db.Delete(key)
		if exists && !deleted {
			t.Error("Expected to delete but it didn't")
		}
	})
}

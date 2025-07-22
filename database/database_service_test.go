package database

import (
	"container/heap"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"
)

type createCall struct {
	value string // value for the Create
	ttl   int64  // TTL for the Create
	index int    // The expected index in the final post-creation TTL increasing order
}
type putCall struct {
	key   string // key for the Put
	value string // value for the Put
	ttl   int64  // TTL for the Put
}

// SetupHelper will take functions and use them to create a database
func SetupHelper(i *InMemoryDatabase, functions *[]any, expectedOrder *map[string]int) {
	for _, function := range *functions {
		switch function.(type) {
		case *createCall:
			arguments := struct {
				Value string `json:"value"`
				Ttl   *int64 `json:"ttl"`
			}{
				function.(*createCall).value,
				&function.(*createCall).ttl,
			}
			_, uuid := i.Create(arguments)
			if expectedOrder != nil {
				(*expectedOrder)[uuid] = function.(*createCall).index
			}
		case *putCall:
			arguments := struct {
				Key   string `json:"key"`
				Value string `json:"value"`
				Ttl   *int64 `json:"ttl"`
			}{
				function.(*putCall).key,
				function.(*putCall).value,
				&function.(*putCall).ttl,
			}
			i.Put(arguments)
		}
	}
}

func TestInMemoryDatabase_Create(t *testing.T) {
	type test []struct {
		value       string // The value for the Create
		ttl         *int64 // The ttl for the Create
		loadedValue string // What value should be loaded after the Create
	}

	tests := []struct {
		name  string
		cases test
	}{
		{
			name: "Create a single new entry",
			cases: test{
				{
					value:       "value",
					loadedValue: "value",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i, err := NewInMemoryDatabase()
			if err != nil {
				t.Error(err)
			}

			for _, testCase := range tt.cases {
				data := struct {
					Value string `json:"value"`
					Ttl   *int64 `json:"ttl"`
				}{
					Value: testCase.value,
					Ttl:   testCase.ttl,
				}

				_, key := i.Create(data)
				val, loaded := i.load(key)
				if val.value != testCase.loadedValue {
					t.Errorf("Error loading value: Create() = %v, want %v where loaded = %v", val, testCase.loadedValue, loaded)
				}
			}
		})
	}
}

func TestInMemoryDatabase_Get(t *testing.T) {
	type test []struct {
		key        string // The key for the Get
		wantValue  string // The expected value for the Get
		wantLoaded bool   // True if it should return a value and false otherwise
		addTTL     bool   // Whether to add a TTL to the Put or not
		ttl        int64  // The ttl for the Put
	}

	tests := []struct {
		name  string
		cases test
	}{
		{
			name: "Get an existing entry with no TTL",
			cases: test{
				{
					key:        "key",
					wantValue:  "value",
					wantLoaded: true,
					addTTL:     false,
				},
			},
		},
		{
			name: "Get an existing entry with valid TTL",
			cases: test{
				{
					key:        "key",
					wantValue:  "value",
					wantLoaded: true,
					addTTL:     true,
					ttl:        100,
				},
			},
		},
		{
			name: "Get an existing entry with expired TTL",
			cases: test{
				{
					key:        "key",
					wantValue:  "value",
					wantLoaded: false,
					addTTL:     true,
					ttl:        -1,
				},
			},
		},
		{
			name: "Get a non-existing entry",
			cases: test{
				{
					key:        "yo",
					wantValue:  "",
					wantLoaded: false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, testCase := range tt.cases {
				i, err := NewInMemoryDatabase()
				if err != nil {
					t.Error(err)
				}

				var ttl *int64
				if testCase.addTTL {
					ttl = &testCase.ttl
				}
				i.Put(struct {
					Key   string `json:"key"`
					Value string `json:"value"`
					Ttl   *int64 `json:"ttl"`
				}{
					Key:   "key",
					Value: "value",
					Ttl:   ttl,
				})

				val, loaded := i.Get(testCase.key)
				if loaded != testCase.wantLoaded {
					t.Errorf("Get() = %v, want %v for whether it was loaded or not", loaded, testCase.wantLoaded)
				}
				if testCase.wantLoaded && (loaded == false || val != testCase.wantValue) {
					t.Errorf("Get() = %v, want %v for loaded value", val, testCase.wantValue)
				}
			}
		})
	}
}

func TestInMemoryDatabase_Put(t *testing.T) {
	type test []struct {
		key         string // key for Put
		value       string // value for Put
		ttl         *int64 // TTL for Put
		want        bool   // True if it should be updated and false if it should be created
		loadedValue string // The value that should be loaded after the Put
	}

	tests := []struct {
		name  string
		cases test
	}{
		{
			name: "Put an existing entry",
			cases: test{
				{
					key:         "key",
					value:       "value",
					want:        false,
					loadedValue: "value",
				},
				{
					key:         "key",
					value:       "value",
					want:        true,
					loadedValue: "value",
				},
			},
		},
		{
			name: "Put a non-existing entry",
			cases: test{
				{
					key:         "dog",
					value:       "hello",
					want:        false,
					loadedValue: "hello",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i, err := NewInMemoryDatabase()
			if err != nil {
				t.Error(err)
			}

			for _, testCase := range tt.cases {
				data := struct {
					Key   string `json:"key"`
					Value string `json:"value"`
					Ttl   *int64 `json:"ttl"`
				}{
					Key:   testCase.key,
					Value: testCase.value,
					Ttl:   testCase.ttl,
				}
				if loaded := i.Put(data); loaded != testCase.want {
					t.Errorf("Put() = %v, want %v", loaded, testCase.want)
				}

				val, loaded := i.load(testCase.key)
				if val.value != testCase.loadedValue {
					t.Errorf("Error loading value: Put() = %v, want %v where loaded = %v", val, testCase.loadedValue, loaded)
				}
			}
		})
	}
}

func TestInMemoryDatabase_Delete(t *testing.T) {
	type test []struct {
		key  string // key for delete
		want bool   // Expectation for whether it was deleted or not
	}

	tests := []struct {
		name  string
		cases test
	}{
		{
			name: "Delete an existing entry",
			cases: test{
				{
					key:  "key",
					want: true,
				},
			},
		},
		{
			name: "Delete a non-existing entry",
			cases: test{
				{
					key:  "yo",
					want: false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i, err := NewInMemoryDatabase()
			if err != nil {
				t.Error(err)
			}

			i.Put(struct {
				Key   string `json:"key"`
				Value string `json:"value"`
				Ttl   *int64 `json:"ttl"`
			}{
				Key:   "key",
				Value: "value",
			})
			for _, testCase := range tt.cases {
				if loaded := i.Delete(testCase.key); loaded != testCase.want {
					t.Errorf("Delete() = %v, want %v", loaded, testCase.want)
				}
			}
		})
	}
}

func TestInMemoryDatabase_GetTTL(t *testing.T) {
	type test []struct {
		key        string // key for get
		wantLoaded bool   // Expected loaded
		wantNil    bool   // Whether the returned TTL pointer should be nil or not
		delay      int64  // How long to delay before sending a GetTTL call
	}

	tests := []struct {
		name  string
		cases test
	}{
		{
			name: "Get an existing entry",
			cases: test{
				{
					key:        "key",
					delay:      2,
					wantLoaded: true,
					wantNil:    false,
				},
			},
		},
		{
			name: "Get a non-existing entry",
			cases: test{
				{
					key:        "yo",
					delay:      0,
					wantLoaded: false,
					wantNil:    true,
				},
			},
		},
		{
			name: "Get an expired entry",
			cases: test{
				{
					key:        "yo",
					delay:      0,
					wantLoaded: false,
					wantNil:    true,
				},
			},
		},
		{
			name: "Get an entry with no expiration",
			cases: test{
				{
					key:        "noExpire",
					delay:      0,
					wantLoaded: true,
					wantNil:    true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ttl int64 = 100
			i, err := NewInMemoryDatabase()
			if err != nil {
				t.Error(err)
			}

			// Add an entry with no expiration
			i.Put(struct {
				Key   string `json:"key"`
				Value string `json:"value"`
				Ttl   *int64 `json:"ttl"`
			}{
				Key:   "noExpire",
				Value: "value",
			})

			for _, testCase := range tt.cases {
				// Add an entry with a ttl of 100
				i.Put(struct {
					Key   string `json:"key"`
					Value string `json:"value"`
					Ttl   *int64 `json:"ttl"`
				}{
					Key:   "key",
					Value: "value",
					Ttl:   &ttl,
				})

				// Wait
				<-time.After(time.Duration(testCase.delay) * time.Second)

				val, loaded := i.GetTTL(testCase.key)
				if loaded != testCase.wantLoaded {
					t.Errorf("Get() = %v, %v", testCase.wantLoaded, loaded)
				}

				if testCase.wantNil {
					if val != nil {
						t.Errorf("Get() = %v, want nil", val)
					}
				} else {
					if val == nil {
						t.Error("Get() = nil, want not nil")
					} else if *val < testCase.delay {
						t.Errorf("Get() = %v, want %v", *val, testCase.delay)
					}
				}
			}
		})
	}
}

func TestInMemoryDatabase_Heap(t *testing.T) {
	tests := []struct {
		name          string
		numKeys       int
		functions     []any
		expectedOrder map[string]int
	}{
		{
			name: "Create Only",
			functions: []any{
				&createCall{"hello5", 50, 4},
				&createCall{"hello1", 10, 0},
				&createCall{"hello3", 30, 2},
				&createCall{"hello4", 40, 3},
				&createCall{"hello2", 20, 1},
			},
			expectedOrder: map[string]int{},
		},
		{
			name: "Put Only",
			functions: []any{
				&putCall{"hello1", "hello1", 10},
				&putCall{"hello2", "hello2", 20},
				&putCall{"hello3", "hello3", 30},
				&putCall{"hello4", "hello4", 40},
				&putCall{"hello3", "hello3", 50},
			},
			expectedOrder: map[string]int{
				"hello1": 0,
				"hello2": 1,
				"hello4": 2,
				"hello3": 3,
			},
		},
		{
			name: "Create Plus Put",
			functions: []any{
				&createCall{"hello1", 10, 0},
				&createCall{"hello2", 20, 1},
				&putCall{"hello3", "hello3", 30},
				&putCall{"hello4", "hello4", 40},
				&putCall{"hello3", "hello3", 50},
			},
			expectedOrder: map[string]int{
				"hello4": 2,
				"hello3": 3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i, err := NewInMemoryDatabase()
			if err != nil {
				t.Error(err)
			}

			SetupHelper(i, &tt.functions, &tt.expectedOrder)

			// Get all ttlHeap information
			i.mu.Lock()
			var copyHeap []ttlHeapData
			for _, data := range *i.ttl {
				key := data.key
				dbEntry, loaded := i.load(key)
				if loaded && *dbEntry.ttl == data.ttl {
					copyHeap = append(copyHeap, data)
				}
			}
			i.mu.Unlock()

			// Sort ttlHeap in decreasing order by the TTL value
			sort.Slice(copyHeap, func(i, j int) bool {
				return copyHeap[i].ttl < copyHeap[j].ttl
			})

			// Make sure the right number of values is stored
			if len(copyHeap) != len(tt.expectedOrder) {
				t.Errorf("Expected copyHeap size to be %v got %v", len(tt.expectedOrder), len(copyHeap))
			}

			// Check the actual order
			for i, actual := range copyHeap {
				expectedIndex, ok := tt.expectedOrder[actual.key]
				if !ok {
					t.Fatalf("key %v is in copyHeap but does not exist in expectedOrder", actual.key)
				}
				if expectedIndex != i {
					t.Errorf("Got key %v at index %v instead of %v", actual.key, expectedIndex, i)
				}
			}
		})
	}
}

func TestInMemoryDatabase_Cleanup(t *testing.T) {
	type checkDeleted struct {
		delay   int64 // Time after initialization to check
		numLeft int   // How many should remain
	}

	tests := []struct {
		name      string
		numKeys   int
		functions []any
		check     []checkDeleted
		final     int64
	}{
		{
			name: "Create Plus Put",
			functions: []any{
				&createCall{"hello1", 1, -1},
				&createCall{"hello2", 2, -1},
				&putCall{"hello3", "hello3", 3},
				&putCall{"hello4", "hello4", 4},
				&putCall{"hello3", "hello3", 5},
			},
			check: []checkDeleted{
				{1, 3},
				{2, 2},
				{3, 2},
				{4, 1},
				{5, 0},
			},
			final: 6,
		},
		{
			name: "Create Only",
			functions: []any{
				&createCall{"hello1", 1, -1},
				&createCall{"hello2", 2, -1},
			},
			check: []checkDeleted{
				{1, 1},
				{2, 0},
			},
			final: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i, err := NewInMemoryDatabase()
			if err != nil {
				t.Error(err)
			}

			SetupHelper(i, &tt.functions, nil)

			timeAfterCreation := time.Now().Unix()

			i.mu.RLock()
			if len(i.database) == 0 {
				t.Errorf("Store is empty")
			}
			i.mu.RUnlock()

			// Check all deletions occur
			for c := range tt.check {
				next := tt.check[c].delay
				delay := next - (time.Now().Unix() - timeAfterCreation)
				if delay > 0 {
					<-time.After(time.Duration(delay)*time.Second + 500*time.Millisecond)
				}

				i.mu.Lock()

				// Check the number of remaining entries
				if len(i.database) != len(*i.ttl) {
					t.Errorf("Expected %v left but got %v. Len(ttlHeap) = %v", tt.check[c].numLeft, len(i.database), len(*i.ttl))
				}

				i.mu.Unlock()
			}
		})
	}
}

func TestInMemoryDatabase_Persistence(t *testing.T) {
	intPtr := func(v int64) *int64 {
		return &v
	}

	tests := []struct {
		name        string
		functions   []any
		expectedDB  dbStore
		expectedTTL *ttlHeap
	}{
		{
			name: "Test saving database",
			functions: []any{
				&putCall{"hello1", "hello1", 10},
				&putCall{"hello2", "hello2", 20},
				&putCall{"hello3", "hello3", 30},
				&putCall{"hello4", "hello4", 40},
				&putCall{"hello3", "hello3", 50},
			},
			expectedDB: dbStore{
				"hello1": databaseEntry{
					value: "hello1",
					ttl:   intPtr(10),
				},
				"hello2": databaseEntry{
					value: "hello2",
					ttl:   intPtr(20),
				},
				"hello3": databaseEntry{
					value: "hello3",
					ttl:   intPtr(50),
				},
				"hello4": databaseEntry{
					value: "hello4",
					ttl:   intPtr(40),
				},
			},
			expectedTTL: &ttlHeap{
				ttlHeapData{
					key: "hello1",
					ttl: 10,
				},
				ttlHeapData{
					key: "hello2",
					ttl: 20,
				},
				ttlHeapData{
					key: "hello3",
					ttl: 50,
				},
				ttlHeapData{
					key: "hello4",
					ttl: 40,
				},
			},
		},
	}

	waitTime := 2 * time.Second

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp := filepath.Join(t.TempDir(), "persist.json") // For persistence

			heap.Init(tt.expectedTTL)

			i, err := NewInMemoryDatabase(WithPersistence(), WithPersistencePeriod(1*time.Second), WithPersistenceOutput(fp))
			if err != nil {
				t.Error(err)
			}
			SetupHelper(i, &tt.functions, nil)

			<-time.After(waitTime)

			data, err := os.ReadFile(fp)
			if err != nil {
				t.Fatal("Failed to read persist.json")
			}

			var db *InMemoryDatabase

			err = json.Unmarshal(data, &db)
			if err != nil {
				t.Fatal("Failed to unmarshal persist.json")
			}

			if !reflect.DeepEqual(db.ttl, i.ttl) {
				t.Errorf("Actual ttl heap does not match persist.json")
			}

			if !reflect.DeepEqual(db.database, i.database) {
				t.Errorf("Actual database does not match persist.json")
			}
		})
	}
}

func TestInMemoryDatabase_StartJson(t *testing.T) {
	tests := []struct {
		name string
		file string
	}{
		{
			name: "Test starting database with json",
			file: "testStartup.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i, err := NewInMemoryDatabase(WithInitialData(tt.file))
			if err != nil {
				t.Error(err)
			}

			data, err := os.ReadFile(tt.file)
			if err != nil {
				t.Errorf("Failed to read %v", tt.file)
			}

			var db *InMemoryDatabase

			err = json.Unmarshal(data, &db)
			if err != nil {
				t.Errorf("Failed to unmarshal %v", tt.file)
			}

			if !reflect.DeepEqual(db.ttl, i.ttl) {
				t.Errorf("Actual ttl heap does not match %v", tt.file)
			}

			if !reflect.DeepEqual(db.database, i.database) {
				t.Errorf("Actual database does not match %v", tt.file)
			}
		})
	}
}

package database

import (
	"sort"
	"testing"
	"time"
)

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
			i := NewInMemoryDatabase()
			for _, testCase := range tt.cases {
				data := struct {
					Value string `json:"value"`
					Ttl   *int64 `json:"ttl"`
				}{
					Value: testCase.value,
					Ttl:   testCase.ttl,
				}

				_, key := i.Create(data)
				val, loaded := i.store.Load(key)
				if val.(databaseEntry).value != testCase.loadedValue {
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
				i := NewInMemoryDatabase()
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
		key         string // Key for Put
		value       string // Value for Put
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
			i := NewInMemoryDatabase()
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

				val, loaded := i.store.Load(testCase.key)
				if val.(databaseEntry).value != testCase.loadedValue {
					t.Errorf("Error loading value: Put() = %v, want %v where loaded = %v", val, testCase.loadedValue, loaded)
				}
			}
		})
	}
}

func TestInMemoryDatabase_Delete(t *testing.T) {
	type test []struct {
		key  string // Key for delete
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
			i := NewInMemoryDatabase()
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
		key        string // Key for get
		wantValue  int64  // Expected TTL
		wantLoaded bool   // Expected loaded
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
					wantValue:  100 + time.Now().Unix(),
					wantLoaded: true,
				},
			},
		},
		{
			name: "Get a non-existing entry",
			cases: test{
				{
					key:        "yo",
					wantValue:  0,
					wantLoaded: false,
				},
			},
		},
		{
			name: "Get an expired entry",
			cases: test{
				{
					key:        "yo",
					wantValue:  0,
					wantLoaded: false,
				},
			},
		},
		{
			name: "Get an entry with no expiration",
			cases: test{
				{
					key:        "noExpire",
					wantValue:  0,
					wantLoaded: false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ttl int64 = 100
			i := NewInMemoryDatabase()

			// Add an entry with an expiration
			i.Put(struct {
				Key   string `json:"key"`
				Value string `json:"value"`
				Ttl   *int64 `json:"ttl"`
			}{
				Key:   "key",
				Value: "value",
				Ttl:   &ttl,
			})

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
				if val, loaded := i.GetTTL(testCase.key); loaded != testCase.wantLoaded || val < testCase.wantValue {
					t.Errorf("Get() = %v, %v, want >=%v, %v", testCase.wantValue, testCase.wantLoaded, val, loaded)
				}
			}
		})
	}
}

func TestInMemoryDatabase_Heap(t *testing.T) {
	type createCall struct {
		value string // Value for the Create
		ttl   int64  // TTL for the Create
		index int    // The expected index in the final post-creation TTL increasing order
	}
	type putCall struct {
		key   string // Key for the Put
		value string // Value for the Put
		ttl   int64  // TTL for the Put
	}

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
			i := NewInMemoryDatabase()
			for _, function := range tt.functions {
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
					tt.expectedOrder[uuid] = function.(*createCall).index
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

			// Get all ttlHeap information
			var copyHeap []ttlHeapData
			for _, data := range *i.ttl {
				key := data.key
				dbEntry, loaded := i.store.Load(key)
				if loaded && *dbEntry.(databaseEntry).ttl == data.ttl {
					copyHeap = append(copyHeap, data)
				}
			}

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
					t.Fatalf("Key %v is in copyHeap but does not exist in expectedOrder", actual.key)
				}
				if expectedIndex != i {
					t.Errorf("Got key %v at index %v instead of %v", actual.key, expectedIndex, i)
				}
			}
		})
	}
}

func TestInMemoryDatabase_Cleanup(t *testing.T) {
	type createCall struct {
		value string // Value for the Create
		ttl   int64  // TTL for the Create
	}
	type putCall struct {
		key   string // Key for the Put
		value string // Value for the Put
		ttl   int64  // TTL for the Put
	}

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
				&createCall{"hello1", 1},
				&createCall{"hello2", 2},
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
				&createCall{"hello1", 1},
				&createCall{"hello2", 2},
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
			i := NewInMemoryDatabase()
			for _, function := range tt.functions {
				switch function.(type) {
				case *createCall:
					arguments := struct {
						Value string `json:"value"`
						Ttl   *int64 `json:"ttl"`
					}{
						function.(*createCall).value,
						&function.(*createCall).ttl,
					}
					i.Create(arguments)
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
			timeAfterCreation := time.Now().Unix()

			// Get initial count
			var count int
			i.store.Range(func(k, v interface{}) bool {
				count++
				return true
			})

			t.Logf("Number of keys: %v", count)
			if count == 0 {
				t.Errorf("Store is empty")
			}

			// Check all deletions occur
			for c := range tt.check {
				next := tt.check[c].delay
				delay := next - (time.Now().Unix() - timeAfterCreation)
				if delay > 0 {
					<-time.After(time.Duration(delay)*time.Second + 500*time.Millisecond)
				}

				i.mu.Lock()

				// Get number of remaining entries
				var count int
				i.store.Range(func(k, v interface{}) bool {
					count++
					return true
				})

				if count != len(*i.ttl) {
					t.Errorf("Expected %v left but got %v. Len(ttlHeap) = %v", tt.check[c].numLeft, count, len(*i.ttl))
				}

				i.mu.Unlock()
			}
		})
	}
}

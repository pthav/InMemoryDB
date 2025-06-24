package database

import (
	"sort"
	"testing"
	"time"
)

func TestInMemoryDatabase_Create(t *testing.T) {
	type test []struct {
		value       string
		ttl         *int64
		want        bool
		loadedValue string
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
					want:        true,
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

				loaded, key := i.Create(data)
				if loaded != testCase.want {
					t.Errorf("Create() = %v, want %v", loaded, testCase.want)
				}

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
		key        string
		wantValue  string
		wantLoaded bool
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
					wantValue:  "value",
					wantLoaded: true,
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
				if val, loaded := i.Get(testCase.key); loaded != testCase.wantLoaded || val != testCase.wantValue {
					t.Errorf("Get() = %v + %v, want %v + %v", testCase.wantValue, testCase.wantLoaded, val, loaded)
				}
			}
		})
	}
}

func TestInMemoryDatabase_Update(t *testing.T) {
	type test []struct {
		key         string
		value       string
		ttl         *int64
		want        bool
		loadedValue string
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
		key  string
		want bool
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
		key        string
		wantValue  int64
		wantLoaded bool
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ttl int64 = 100
			i := NewInMemoryDatabase()
			i.Put(struct {
				Key   string `json:"key"`
				Value string `json:"value"`
				Ttl   *int64 `json:"ttl"`
			}{
				Key:   "key",
				Value: "value",
				Ttl:   &ttl,
			})
			for _, testCase := range tt.cases {
				if val, loaded := i.GetTTL(testCase.key); loaded != testCase.wantLoaded || val < testCase.wantValue {
					t.Errorf("Get() = %v + %v, want >=%v + %v", testCase.wantValue, testCase.wantLoaded, val, loaded)
				}
			}
		})
	}
}

func TestInMemoryDatabase_Heap(t *testing.T) {
	type createCall struct {
		value string
		ttl   int64
		index int
	}
	type updateCall struct {
		key   string
		value string
		ttl   int64
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
				&updateCall{"hello1", "hello1", 10},
				&updateCall{"hello2", "hello2", 20},
				&updateCall{"hello3", "hello3", 30},
				&updateCall{"hello4", "hello4", 40},
				&updateCall{"hello3", "hello3", 50},
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
				&updateCall{"hello3", "hello3", 30},
				&updateCall{"hello4", "hello4", 40},
				&updateCall{"hello3", "hello3", 50},
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
				case *updateCall:
					arguments := struct {
						Key   string `json:"key"`
						Value string `json:"value"`
						Ttl   *int64 `json:"ttl"`
					}{
						function.(*updateCall).key,
						function.(*updateCall).value,
						&function.(*updateCall).ttl,
					}
					i.Put(arguments)
				}
			}

			var copyHeap []ttlHeapData
			for _, data := range *i.ttl {
				key := data.key
				dbEntry, loaded := i.store.Load(key)
				if loaded && dbEntry.(databaseEntry).ttl == data.ttl {
					copyHeap = append(copyHeap, data)
				}
			}

			sort.Slice(copyHeap, func(i, j int) bool {
				return copyHeap[i].ttl < copyHeap[j].ttl
			})

			if len(copyHeap) != len(tt.expectedOrder) {
				t.Errorf("Expected copyHeap size to be %v got %v", len(tt.expectedOrder), len(copyHeap))
			}

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

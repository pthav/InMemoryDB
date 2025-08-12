package database

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
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

type deleteCall struct {
	key string // key for the Delete
}

// setupHelper will take functions and use them to create a database
func setupHelper(i *InMemoryDatabase, functions *[]any, expectedOrder *map[string]int) {
	for _, function := range *functions {
		switch function.(type) {
		case *createCall:
			arguments := struct {
				Value string `json:"value"`
				Ttl   *int64 `json:"ttl"`
			}{
				Value: function.(*createCall).value,
			}
			if function.(*createCall).ttl >= 0 {
				arguments.Ttl = &function.(*createCall).ttl
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
				Key:   function.(*putCall).key,
				Value: function.(*putCall).value,
			}
			if function.(*putCall).ttl >= 0 {
				arguments.Ttl = &function.(*putCall).ttl
			}

			i.Put(arguments)
		case *deleteCall:
			i.Delete(function.(*deleteCall).key)
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

func TestInMemoryDatabase_Cleanup(t *testing.T) {
	type checkDeleted struct {
		delay   int64 // Time after initialization to check in milliseconds
		numLeft int   // How many should remain
	}

	tests := []struct {
		name      string
		numKeys   int
		functions []any
		check     []checkDeleted
		unique    int
	}{
		{
			name: "Create Plus Put",
			functions: []any{
				&putCall{"hello4", "hello4", 4},
				&putCall{"hello3", "hello3", 5},
				&putCall{"hello3", "hello3", 3},
				&createCall{"hello1", 1, -1},
				&createCall{"hello2", 2, -1},
			},
			check: []checkDeleted{
				{0, 4},
				{1500, 3},
				{2500, 2},
				{3500, 1},
				{4500, 0},
			},
			unique: 4,
		},
		{
			name: "Create Only",
			functions: []any{
				&createCall{"hello1", 1, -1},
				&createCall{"hello2", 2, -1},
			},
			check: []checkDeleted{
				{0, 2},
				{1500, 1},
				{2500, 0},
			},
			unique: 2,
		},
		{
			name: "Put Only",
			functions: []any{
				&putCall{"hello2", "hello2", 2},
				&putCall{"hello1", "hello1", 1},
			},
			check: []checkDeleted{
				{0, 2},
				{1500, 1},
				{2500, 0},
			},
			unique: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i, err := NewInMemoryDatabase()
			if err != nil {
				t.Error(err)
			}

			setupHelper(i, &tt.functions, nil)

			timeAfterCreation := time.Now().UnixMilli()

			i.mu.RLock()
			if len(i.database) != tt.unique {
				t.Errorf("Store is wrong size")
			}
			i.mu.RUnlock()

			// Check all deletions occur
			for c := range tt.check {
				next := tt.check[c].delay
				delay := next - (time.Now().UnixMilli() - timeAfterCreation)
				if delay > 0 {
					<-time.After(time.Duration(delay) * time.Millisecond)
				}

				i.mu.Lock()

				// Check the number of remaining entries
				if len(i.database) != tt.check[c].numLeft {
					t.Errorf("Expected %v left after %v but got %v. Len(ttlHeap) = %v", tt.check[c].numLeft, next, len(i.database), len(*i.ttl))
				}

				i.mu.Unlock()
			}
		})
	}
}

func TestInMemoryDatabase_Persistence(t *testing.T) {
	tests := []struct {
		name      string
		functions []any
	}{
		{
			name: "Test saving database",
			functions: []any{
				&putCall{"hello1", "hello1", 10},
				&deleteCall{"hello1"},
				&putCall{"hello2", "hello2", 20},
				&putCall{"hello3", "hello3", 30},
				&putCall{"hello4", "hello4", 40},
				&deleteCall{"hello4"},
				&putCall{"hello3", "hello3", 50},
				&deleteCall{"hello20"},
				&putCall{"noTTL", "noTTL", -1},
				&createCall{"hello1", 10, 0},
				&createCall{"hello1", -1, 0}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp := t.TempDir()

			i, err := NewInMemoryDatabase(
				WithAofPersistence(),
				WithAofPersistencePeriod(1*time.Second),
				WithAofPersistenceFile(filepath.Join(fp, "persist-aof")),
				WithDatabasePersistence(),
				WithDatabasePersistencePeriod(1*time.Second),
				WithDatabasePersistenceFile(filepath.Join(fp, "persist-database.json")))
			if err != nil {
				t.Error(err)
			}
			setupHelper(i, &tt.functions, nil)

			i.Shutdown()

			// Test AOF persistence
			file, err := os.Open(filepath.Join(fp, "persist-aof"))
			if err != nil {
				t.Error(err)
			}
			defer file.Close()

			scanner := bufio.NewScanner(file)
			for i, function := range tt.functions {
				scanner.Scan()
				line := scanner.Text()
				args := strings.Split(line, " ")

				switch function.(type) {
				case *deleteCall:
					if args[0] != "DELETE" {
						t.Errorf("Expected delete command got %v", args[0])
					}

					if len(args) != 2 {
						t.Errorf("For delete function at index %v, got incorrect number of args. Expected %v, but got %v", i, 2, len(args))
					}

					if function.(*deleteCall).key != args[1] {
						t.Errorf("For delete function at index %v, got incorrect key. Expected %v, but got %v", i, function.(*deleteCall).key, args[1])
					}
				case *putCall:
					if args[0] != "PUT" {
						t.Errorf("Expected put command got %v", args[0])
					}

					if len(args) != 4 {
						t.Errorf("For put function at index %v, got incorrect number of args. Expected %v, but got %v", i, 4, len(args))
					}

					if function.(*putCall).key != args[1] {
						t.Errorf("For put function at index %v, got incorrect key. Expected %v, but got %v", i, function.(*putCall).key, args[1])
					}

					if function.(*putCall).value != args[2] {
						t.Errorf("For put function at index %v, got incorrect value. Expected %v, but got %v", i, function.(*putCall).value, args[2])
					}

					if strconv.Itoa(int(function.(*putCall).ttl)) != args[3] {
						t.Errorf("For put function at index %v, got incorrect ttl. Expected %v, but got %v", i, function.(*putCall).ttl, args[3])
					}
				case *createCall:
					if args[0] != "PUT" {
						t.Errorf("Expected put command got %v", args[0])
					}

					if len(args) != 4 {
						t.Errorf("For create function at index %v, got incorrect number of args. Expected %v, but got %v", i, 4, len(args))
					}

					if function.(*createCall).value != args[2] {
						t.Errorf("For create function at index %v, got incorrect value. Expected %v, but got %v", i, function.(*createCall).value, args[2])
					}

					if strconv.Itoa(int(function.(*createCall).ttl)) != args[3] {
						t.Errorf("For create function at index %v, got incorrect ttl. Expected %v, but got %v", i, function.(*createCall).ttl, args[3])
					}
				}
			}

			// Test database persistence
			data, err := os.ReadFile(filepath.Join(fp, "persist-database.json"))
			if err != nil {
				t.Fatal("Failed to read persistDatabase.json")
			}

			var decodedData *InMemoryDatabase
			dec := gob.NewDecoder(bytes.NewBuffer(data))
			if err := dec.Decode(&decodedData); err != nil {
				log.Fatal("Decode error:", err)
			}

			if !reflect.DeepEqual(decodedData.ttl, i.ttl) {
				t.Errorf("Actual ttl heap does not match persistDatabase.json")
			}

			if !reflect.DeepEqual(decodedData.database, i.database) {
				t.Errorf("Actual database does not match persistDatabase.json")
			}
		})
	}
}

func TestInMemoryDatabase_DatabaseStartJson(t *testing.T) {
	tests := []struct {
		name string
		file string
	}{
		{
			name: "Test starting database with json",
			file: "testDatabaseStartup.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i, err := NewInMemoryDatabase(WithInitialData(tt.file, true))
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

func TestInMemoryDatabase_AofStart(t *testing.T) {
	type expectationCommand struct {
		key    string // Key to GET
		exists bool   // Whether it should exist
		value  string // What value it should have
		ttl    int64  // What TTL it should have
	}

	tests := []struct {
		name     string
		commands []string
		expected []expectationCommand
	}{
		{
			name: "Test starting database with AOF",
			commands: []string{
				"PUT hello1 hello1 -1",
				"PUT hello2 hello2 2751785118",
				"DELETE hello1",
				"PUT hello3 hello3 -1",
				"DELETE doesn'tExist",
			},
			expected: []expectationCommand{
				{
					key:    "hello1",
					exists: false,
				},
				{
					key:    "hello2",
					exists: true,
					value:  "hello2",
					ttl:    2751785118,
				},
				{
					key:    "hello3",
					exists: true,
					value:  "hello3",
					ttl:    -1,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp := t.TempDir()

			file, err := os.Create(filepath.Join(fp, "aof"))
			if err != nil {
				t.Error(err)
			}
			defer file.Close()

			output := ""
			for _, command := range tt.commands {
				output += fmt.Sprintf("%v\n", command)
			}

			_, err = file.WriteString(output)
			if err != nil {
				t.Error(err)
			}

			db, err := NewInMemoryDatabase(WithInitialData(filepath.Join(fp, "aof"), false))
			if err != nil {
				t.Error(err)
			}

			for i, command := range tt.expected {
				value, loaded := db.Get(command.key)
				if loaded != command.exists {
					t.Fatalf("For command at index %v, expected %v but got %v", i, command.exists, loaded)
				}

				if loaded && value != command.value {
					t.Errorf("For command at index %v, expected %v but got %v", i, command.value, value)
				}

				if !command.exists {
					continue
				}

				ttl, _ := db.GetTTL(command.key)
				if command.ttl == -1 && ttl != nil {
					t.Errorf("For command at index %v, expected nil ttl but got %v", i, *ttl)
				} else if command.ttl != -1 && ttl == nil {
					t.Errorf("For command at index %v, expected %v but got nil", i, command.ttl)
				} else if command.ttl != -1 && *ttl != command.ttl-time.Now().Unix() {
					t.Errorf("For command at index %v, expected %v but got %v", i, command.ttl, *ttl)
				}
			}
		})
	}
}

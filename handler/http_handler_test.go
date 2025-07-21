package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

// databaseTestImplementation is an implementation of database used for test cases
type databaseTestImplementation struct {
	createCalls []struct {
		key   string
		value string
		ttl   *int64
	}
	createKey    string
	createReturn bool
	readCalls    []struct {
		key string
	}
	readReturn bool
	readString string
	putCalls   []struct {
		key   string
		value string
		ttl   *int64
	}
	putReturn   bool
	deleteCalls []struct {
		key string
	}
	deleteReturn bool
	getTTLCalls  []struct {
		key string
	}
	getTTLReturn bool
	getTTLTime   *int64
}

func (db *databaseTestImplementation) Create(data struct {
	Value string `json:"value"`
	Ttl   *int64 `json:"ttl"`
}) (bool, string) {
	db.createCalls = append(db.createCalls, struct {
		key   string
		value string
		ttl   *int64
	}{db.createKey, data.Value, data.Ttl})
	return db.createReturn, db.createKey
}

func (db *databaseTestImplementation) Get(key string) (string, bool) {
	db.readCalls = append(db.readCalls, struct {
		key string
	}{key})
	return db.readString, db.readReturn
}

func (db *databaseTestImplementation) Put(data struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Ttl   *int64 `json:"ttl"`
}) bool {
	db.putCalls = append(db.putCalls, struct {
		key   string
		value string
		ttl   *int64
	}{data.Key, data.Value, data.Ttl})
	return db.putReturn
}

func (db *databaseTestImplementation) Delete(key string) bool {
	db.deleteCalls = append(db.deleteCalls, struct {
		key string
	}{key})
	return db.deleteReturn
}

func (db *databaseTestImplementation) GetTTL(key string) (*int64, bool) {
	db.getTTLCalls = append(db.getTTLCalls, struct {
		key string
	}{key})
	return db.getTTLTime, db.getTTLReturn
}

// Helper for making an int pointer from an r-value
func intPtr(v int64) *int64 {
	return &v
}

// testCase is a general test case struct for a majority of these test functions
type testCase struct {
	name         string // Test case name
	key          string // Test case key used
	value        string // Value for the JSON request body
	ttl          *int64 // TTL for the JSON request body
	status       int    // Desired return status
	createReturn bool   // Desired bool return from Create
	readReturn   bool   // Desired bool return from Read
	updateReturn bool   // Desired bool return from Update
	deleteReturn bool   // Desired bool return from Delete
	getTTLReturn bool   // Desired bool return from getTTL
	checkCalls   bool   // Number of expected DB function calls
}

func testHelper(t *testing.T, tt testCase, method string, path string, body string) (*bytes.Buffer, *databaseTestImplementation) {
	// Set up writer and request
	w := httptest.NewRecorder()
	r := &http.Request{
		Method: method,
		URL:    &url.URL{Path: path},
		Body:   io.NopCloser(strings.NewReader(body)),
	}

	// Set up database
	db := &databaseTestImplementation{
		createReturn: tt.createReturn,
		createKey:    tt.key,
		readReturn:   tt.readReturn,
		readString:   tt.value,
		putReturn:    tt.updateReturn,
		deleteReturn: tt.deleteReturn,
		getTTLReturn: tt.getTTLReturn,
		getTTLTime:   tt.ttl,
	}
	h := NewHandler(db, slog.New(slog.DiscardHandler))
	h.ServeHTTP(w, r)

	if w.Code != tt.status {
		t.Errorf("response code = %v; want %v", w.Code, tt.status)
	}

	return w.Body, db
}

func TestWrapper_postHandler(t *testing.T) {
	tests := []testCase{
		{
			name:         "Try to create non-existing key value pair",
			key:          "testKey",
			value:        "testValue",
			ttl:          intPtr(0),
			status:       http.StatusCreated,
			createReturn: true,
			checkCalls:   true,
		},
		{
			name:         "Try to create non-existing key value pair without a TTL",
			key:          "testKey",
			value:        "testValue",
			status:       http.StatusCreated,
			createReturn: true,
			checkCalls:   true,
		},
		{
			name:       "Send a bad request body",
			key:        "testKey",
			value:      `{"test": "test"}`,
			ttl:        intPtr(200),
			status:     http.StatusBadRequest,
			checkCalls: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method := "POST"
			path := "/v1/keys"
			var requestBody string
			if tt.ttl != nil {
				requestBody = fmt.Sprintf(`{"value": "%s", "ttl": %v}`, tt.value, *tt.ttl)
			} else {
				requestBody = fmt.Sprintf(`{"value": "%s"}`, tt.value)
			}
			wBody, db := testHelper(t, tt, method, path, requestBody)

			if tt.checkCalls {
				var body postResponse
				err := json.NewDecoder(wBody).Decode(&body)
				if err != nil {
					t.Errorf("Failed to decode response body JSON: %v", err)
				}

				expected := postResponse{Key: tt.key}

				if !reflect.DeepEqual(expected, body) {
					t.Errorf("response body = %v; want %v", body, expected)
				}

				if len(db.createCalls) == 0 {
					t.Errorf("Create() calls not created")
				}

				if db.createCalls[0].key != tt.key {
					t.Errorf("Create() key = %v; want %v", db.createCalls[0].key, tt.key)
				}

				if db.createCalls[0].value != tt.value {
					t.Errorf("Create() value = %v; want %v", db.createCalls[0].value, tt.value)
				}

				if tt.ttl != nil && *db.createCalls[0].ttl != *tt.ttl {
					t.Errorf("Create() TTL = %v; want %v", db.createCalls[0].ttl, tt.ttl)
				}
			}
		})
	}
}

func TestWrapper_getHandler(t *testing.T) {
	tests := []testCase{
		{
			name:       "Get an existing key value pair",
			key:        "testKey",
			value:      "testValue",
			status:     http.StatusOK,
			readReturn: true,
			checkCalls: true,
		},
		{
			name:       "Try to read a non-existing key value pair",
			key:        "testKey",
			value:      "",
			status:     http.StatusNotFound,
			readReturn: false,
			checkCalls: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method := "GET"
			path := fmt.Sprintf("/v1/keys/%s", tt.key)
			requestBody := ""
			wBody, db := testHelper(t, tt, method, path, requestBody)

			if tt.readReturn {
				var body getResponse
				err := json.NewDecoder(wBody).Decode(&body)
				if err != nil {
					t.Errorf("Failed to decode response body JSON: %v", err)
				}

				expected := getResponse{Key: tt.key, Value: tt.value}

				if !reflect.DeepEqual(expected, body) {
					t.Errorf("response body = %v; want %v", body, expected)
				}
			}

			if tt.checkCalls {
				if len(db.readCalls) == 0 {
					t.Errorf("Get() calls not created")
				}

				if db.readCalls[0].key != tt.key {
					t.Errorf("Get() key = %v; want %v", db.readCalls[0].key, tt.key)
				}
			}
		})
	}
}

func TestWrapper_putHandler(t *testing.T) {
	tests := []testCase{
		{
			name:         "Put non-existing key value pair",
			key:          "testKey",
			value:        "testValue",
			ttl:          intPtr(40),
			status:       http.StatusCreated,
			updateReturn: false,
			checkCalls:   true,
		},
		{
			name:         "Put an existing key value pair",
			key:          "testKey",
			value:        "testValue",
			ttl:          intPtr(100),
			status:       http.StatusOK,
			updateReturn: true,
			checkCalls:   true,
		},
		{
			name:         "Put an existing key value pair without updating TTL",
			key:          "testKey",
			value:        "testValue",
			ttl:          nil,
			status:       http.StatusOK,
			updateReturn: true,
			checkCalls:   true,
		},
		{
			name:       "Send a bad request body",
			key:        "testKey",
			value:      `{"test": "test"}`,
			ttl:        intPtr(500),
			status:     http.StatusBadRequest,
			checkCalls: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method := "PUT"
			path := fmt.Sprintf("/v1/keys/%s", tt.key)
			var requestBody string
			if tt.ttl != nil {
				requestBody = fmt.Sprintf(`{"value": "%s", "ttl": %v}`, tt.value, *tt.ttl)
			} else {
				requestBody = fmt.Sprintf(`{"value": "%s"}`, tt.value)
			}
			_, db := testHelper(t, tt, method, path, requestBody)

			// Check expectations
			if tt.checkCalls {
				if len(db.putCalls) == 0 {
					t.Errorf("Put() calls not created")
				}

				if db.putCalls[0].key != tt.key {
					t.Errorf("Put() key = %v; want %v", db.putCalls[0].key, tt.key)
				}

				if db.putCalls[0].value != tt.value {
					t.Errorf("Put() value = %v; want %v", db.putCalls[0].value, tt.value)
				}

				if tt.ttl != nil && *db.putCalls[0].ttl != *tt.ttl {
					t.Errorf("Put() TTL = %v; want %v", db.putCalls[0].ttl, tt.ttl)
				}
			}
		})
	}
}

func TestWrapper_deleteHandler(t *testing.T) {
	tests := []testCase{
		{
			name:         "Delete an existing key value pair",
			key:          "testKey",
			status:       http.StatusOK,
			deleteReturn: true,
			checkCalls:   true,
		},
		{
			name:         "Try to delete a non-existing key value pair",
			key:          "testKey",
			status:       http.StatusNotFound,
			deleteReturn: false,
			checkCalls:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method := "DELETE"
			path := fmt.Sprintf("/v1/keys/%s", tt.key)
			requestBody := ""
			_, db := testHelper(t, tt, method, path, requestBody)

			if tt.checkCalls {
				if len(db.deleteCalls) == 0 {
					t.Errorf("Delete() calls not created")
				}

				if db.deleteCalls[0].key != tt.key {
					t.Errorf("Delete() key = %v; want %v", db.deleteCalls[0].key, tt.key)
				}
			}
		})
	}
}

func TestWrapper_getTTLHandler(t *testing.T) {
	tests := []testCase{
		{
			name:         "Get an existing key value pair",
			key:          "testKey",
			ttl:          intPtr(100),
			status:       http.StatusOK,
			getTTLReturn: true,
			checkCalls:   true,
		},
		{
			name:         "Get a non-expiring key value pair",
			key:          "testKey",
			ttl:          nil,
			status:       http.StatusOK,
			getTTLReturn: true,
			checkCalls:   true,
		},
		{
			name:         "Try to read a non-existing key value pair",
			key:          "testKey",
			ttl:          intPtr(100),
			status:       http.StatusNotFound,
			getTTLReturn: false,
			checkCalls:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method := "GET"
			path := fmt.Sprintf("/v1/ttl/%s", tt.key)
			requestBody := ""
			wBody, db := testHelper(t, tt, method, path, requestBody)

			if tt.getTTLReturn {
				var body getTTLResponse
				err := json.NewDecoder(wBody).Decode(&body)
				if err != nil {
					t.Errorf("Failed to decode response body JSON: %v", err)
				}

				expected := getTTLResponse{Key: tt.key}
				if tt.ttl != nil {
					expected.TTL = *tt.ttl
				}

				if !reflect.DeepEqual(expected, body) {
					t.Errorf("response body = %v; want %v", body, expected)
				}
			}

			if tt.checkCalls {
				if len(db.getTTLCalls) == 0 {
					t.Errorf("Get() calls not created")
				}

				if db.getTTLCalls[0].key != tt.key {
					t.Errorf("Get() key = %v; want %v", db.getTTLCalls[0].key, tt.key)
				}
			}
		})
	}
}

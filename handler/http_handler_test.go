package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

// DatabaseTestImplementation is an implementation of database used for test cases
type databaseTestImplementation struct {
	createCalls []struct {
		key   string
		value string
		ttl   *int
	}
	createKey    string
	createReturn bool
	readCalls    []struct {
		key string
	}
	readReturn  bool
	readString  string
	updateCalls []struct {
		key   string
		value string
		ttl   *int
	}
	updateReturn bool
	deleteCalls  []struct {
		key string
	}
	deleteReturn bool
}

func (db *databaseTestImplementation) Create(data struct {
	Value string `json:"value"`
	Ttl   *int   `json:"ttl"`
}) (bool, string) {
	db.createCalls = append(db.createCalls, struct {
		key   string
		value string
		ttl   *int
	}{db.createKey, data.Value, data.Ttl})
	return db.createReturn, db.createKey
}

func (db *databaseTestImplementation) Read(key string) (string, bool) {
	db.readCalls = append(db.readCalls, struct {
		key string
	}{key})
	return db.readString, db.readReturn
}

func (db *databaseTestImplementation) Update(data struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Ttl   *int   `json:"ttl"`
}) bool {
	db.updateCalls = append(db.updateCalls, struct {
		key   string
		value string
		ttl   *int
	}{data.Key, data.Value, data.Ttl})
	return db.updateReturn
}

func (db *databaseTestImplementation) Delete(key string) bool {
	db.deleteCalls = append(db.deleteCalls, struct {
		key string
	}{key})
	return db.deleteReturn
}

func TestWrapper_createHandler(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		ttl          int
		status       int
		createReturn bool
		checkCalls   bool
	}{
		{
			name:         "Try to create non-existing key value pair",
			key:          "testKey",
			value:        "testValue",
			ttl:          0,
			status:       http.StatusCreated,
			createReturn: true,
			checkCalls:   true,
		},
		{
			name:       "Send a bad request body",
			key:        "testKey",
			value:      `{"test": "test"}`,
			ttl:        200,
			status:     http.StatusBadRequest,
			checkCalls: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up writer and request
			w := httptest.NewRecorder()
			r := &http.Request{
				Method: "POST",
				URL:    &url.URL{Path: "/v1/keys"},
				Body:   io.NopCloser(strings.NewReader(fmt.Sprintf(`{"key": "%s", "value": "%s", "ttl": %v}`, tt.key, tt.value, tt.ttl))),
			}

			// Set up database
			db := &databaseTestImplementation{
				createCalls: []struct {
					key   string
					value string
					ttl   *int
				}{},
				createReturn: tt.createReturn,
				createKey:    tt.key,
			}
			h := NewHandler(db, slog.New(slog.DiscardHandler))
			h.ServeHTTP(w, r)

			if w.Code != tt.status {
				t.Errorf("response code = %v; want %v", w.Code, tt.status)
			}

			if tt.checkCalls {
				var body createResponse
				err := json.NewDecoder(w.Body).Decode(&body)
				if err != nil {
					t.Errorf("Failed to decode response body JSON: %v", err)
				}

				expected := createResponse{Key: tt.key}

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

				if *db.createCalls[0].ttl != tt.ttl {
					t.Errorf("Create() TTL = %v; want %v", db.createCalls[0].ttl, tt.ttl)
				}
			}
		})
	}
}

func TestWrapper_readHandler(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		value      string
		status     int
		readReturn bool
		checkCalls bool
	}{
		{
			name:       "Read an existing key value pair",
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
			// Set up writer and request
			w := httptest.NewRecorder()
			r := &http.Request{
				Method: "GET",
				URL:    &url.URL{Path: fmt.Sprintf("/v1/keys/%s", tt.key)},
			}

			// Set up database
			db := &databaseTestImplementation{
				readCalls: []struct {
					key string
				}{},
				readReturn: tt.readReturn,
				readString: tt.value,
			}
			h := NewHandler(db, slog.New(slog.DiscardHandler))
			h.ServeHTTP(w, r)

			// Check expectations
			if w.Code != tt.status {
				t.Errorf("response code = %v; want %v", w.Code, tt.status)
			}

			var body readResponse
			err := json.NewDecoder(w.Body).Decode(&body)
			if err != nil {
				t.Errorf("Failed to decode response body JSON: %v", err)
			}

			expected := readResponse{Key: tt.key, Value: tt.value}

			if !reflect.DeepEqual(expected, body) {
				t.Errorf("response body = %v; want %v", body, expected)
			}

			if tt.checkCalls {
				if len(db.readCalls) == 0 {
					t.Errorf("Read() calls not created")
				}

				if db.readCalls[0].key != tt.key {
					t.Errorf("Read() key = %v; want %v", db.readCalls[0].key, tt.key)
				}
			}
		})
	}
}

func TestWrapper_updateHandler(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		ttl          int
		status       int
		updateReturn bool
		checkCalls   bool
	}{
		{
			name:         "Update non-existing key value pair",
			key:          "testKey",
			value:        "testValue",
			ttl:          40,
			status:       http.StatusCreated,
			updateReturn: false,
			checkCalls:   true,
		},
		{
			name:         "Update an existing key value pair",
			key:          "testKey",
			value:        "testValue",
			ttl:          100,
			status:       http.StatusOK,
			updateReturn: true,
			checkCalls:   true,
		},
		{
			name:       "Send a bad request body",
			key:        "testKey",
			value:      `{"test": "test"}`,
			ttl:        500,
			status:     http.StatusBadRequest,
			checkCalls: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up writer and request
			w := httptest.NewRecorder()
			r := &http.Request{
				Method: "PUT",
				URL:    &url.URL{Path: fmt.Sprintf("/v1/keys/%s", tt.key)},
				Body:   io.NopCloser(strings.NewReader(fmt.Sprintf(`{"value": "%s", "ttl": %v}`, tt.value, tt.ttl))),
			}

			// Set up database
			db := &databaseTestImplementation{
				updateCalls: []struct {
					key   string
					value string
					ttl   *int
				}{},
				updateReturn: tt.updateReturn,
			}
			h := NewHandler(db, slog.New(slog.DiscardHandler))
			h.ServeHTTP(w, r)

			if w.Code != tt.status {
				t.Errorf("response code = %v; want %v", w.Code, tt.status)
			}

			// Check expectations
			if tt.checkCalls {
				if len(db.updateCalls) == 0 {
					t.Errorf("Update() calls not created")
				}

				if db.updateCalls[0].key != tt.key {
					t.Errorf("Update() key = %v; want %v", db.updateCalls[0].key, tt.key)
				}

				if db.updateCalls[0].value != tt.value {
					t.Errorf("Update() value = %v; want %v", db.updateCalls[0].value, tt.value)
				}

				if *db.updateCalls[0].ttl != tt.ttl {
					t.Errorf("Update() TTL = %v; want %v", db.updateCalls[0].ttl, tt.ttl)
				}
			}
		})
	}
}

func TestWrapper_deleteHandler(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		status       int
		deleteReturn bool
		checkCalls   bool
	}{
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
			// Set up writer and request
			w := httptest.NewRecorder()
			r := &http.Request{
				Method: "DELETE",
				URL:    &url.URL{Path: fmt.Sprintf("/v1/keys/%s", tt.key)},
			}

			// Set up database
			db := &databaseTestImplementation{
				deleteCalls: []struct {
					key string
				}{},
				deleteReturn: tt.deleteReturn,
			}
			h := NewHandler(db, slog.New(slog.DiscardHandler))
			h.ServeHTTP(w, r)

			// Check expectations
			if w.Code != tt.status {
				t.Errorf("response code = %v; want %v", w.Code, tt.status)
			}

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

func TestLoggingMiddleware(t *testing.T) {
	// Create logger
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))
	wrapper := Wrapper{logger: logger}

	router := mux.NewRouter()
	router.Use(wrapper.loggingMiddleware)
	router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("handler reached"))
		if err != nil {
			t.Errorf("Error writing response: %v", err)
		}
	})

	// Serve test requests
	r := httptest.NewRequest("GET", "/test", io.NopCloser(strings.NewReader(`{"key":"test","value":"test"}`)))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	// Check expectations
	if status := w.Code; status != http.StatusOK {
		t.Errorf("unexpected status: got %v, want %v", status, http.StatusOK)
	}

	var logLine map[string]any
	err := json.Unmarshal([]byte(logBuffer.String()), &logLine)
	if err != nil {
		t.Errorf("Error unmarshalling log: %v", err)
	}
	expectedBody := map[string]any{
		"key":   "test",
		"value": "test",
	}

	if reflect.DeepEqual(logLine["Body"], expectedBody) == false {
		t.Errorf("Body does not match expected body: got %v, want %v", logLine, expectedBody)
	}

	if logLine["method"] != "GET" {
		t.Errorf("log equals %v, should contain %v", logBuffer.String(), `GET`)
	}
}

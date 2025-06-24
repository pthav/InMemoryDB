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
	getTTLTime   int64
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

func (db *databaseTestImplementation) GetTTL(key string) (int64, bool) {
	db.getTTLCalls = append(db.getTTLCalls, struct {
		key string
	}{key})
	return db.getTTLTime, db.getTTLReturn
}

func TestWrapper_postHandler(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		ttl          int64
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
				Body:   io.NopCloser(strings.NewReader(fmt.Sprintf(`{"value": "%s", "ttl": %v}`, tt.value, tt.ttl))),
			}

			// Set up database
			db := &databaseTestImplementation{
				createCalls: []struct {
					key   string
					value string
					ttl   *int64
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
				var body postResponse
				err := json.NewDecoder(w.Body).Decode(&body)
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

				if *db.createCalls[0].ttl != tt.ttl {
					t.Errorf("Create() TTL = %v; want %v", db.createCalls[0].ttl, tt.ttl)
				}
			}
		})
	}
}

func TestWrapper_getHandler(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		value      string
		status     int
		readReturn bool
		checkCalls bool
	}{
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

			var body getResponse
			err := json.NewDecoder(w.Body).Decode(&body)
			if err != nil {
				t.Errorf("Failed to decode response body JSON: %v", err)
			}

			expected := getResponse{Key: tt.key, Value: tt.value}

			if !reflect.DeepEqual(expected, body) {
				t.Errorf("response body = %v; want %v", body, expected)
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
	tests := []struct {
		name         string
		key          string
		value        string
		ttl          int64
		status       int
		updateReturn bool
		checkCalls   bool
	}{
		{
			name:         "Put non-existing key value pair",
			key:          "testKey",
			value:        "testValue",
			ttl:          40,
			status:       http.StatusCreated,
			updateReturn: false,
			checkCalls:   true,
		},
		{
			name:         "Put an existing key value pair",
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
				putCalls: []struct {
					key   string
					value string
					ttl   *int64
				}{},
				putReturn: tt.updateReturn,
			}
			h := NewHandler(db, slog.New(slog.DiscardHandler))
			h.ServeHTTP(w, r)

			if w.Code != tt.status {
				t.Errorf("response code = %v; want %v", w.Code, tt.status)
			}

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

				if *db.putCalls[0].ttl != tt.ttl {
					t.Errorf("Put() TTL = %v; want %v", db.putCalls[0].ttl, tt.ttl)
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

func TestWrapper_getTTLHandler(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		ttl          int64
		status       int
		getTTLReturn bool
		checkCalls   bool
	}{
		{
			name:         "Get an existing key value pair",
			key:          "testKey",
			ttl:          100,
			status:       http.StatusOK,
			getTTLReturn: true,
			checkCalls:   true,
		},
		{
			name:         "Try to read a non-existing key value pair",
			key:          "testKey",
			ttl:          100,
			status:       http.StatusNotFound,
			getTTLReturn: false,
			checkCalls:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up writer and request
			w := httptest.NewRecorder()
			r := &http.Request{
				Method: "GET",
				URL:    &url.URL{Path: fmt.Sprintf("/v1/ttl/%s", tt.key)},
			}

			// Set up database
			db := &databaseTestImplementation{
				getTTLCalls: []struct {
					key string
				}{},
				getTTLReturn: tt.getTTLReturn,
				getTTLTime:   tt.ttl,
			}
			h := NewHandler(db, slog.New(slog.DiscardHandler))
			h.ServeHTTP(w, r)

			// Check expectations
			if w.Code != tt.status {
				t.Errorf("response code = %v; want %v", w.Code, tt.status)
			}

			var body getTTLResponse
			err := json.NewDecoder(w.Body).Decode(&body)
			if err != nil {
				t.Errorf("Failed to decode response body JSON: %v", err)
			}

			expected := getTTLResponse{Key: tt.key, TTL: tt.ttl}

			if !reflect.DeepEqual(expected, body) {
				t.Errorf("response body = %v; want %v", body, expected)
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

func TestJsonValidationPost(t *testing.T) {
	t.Run("Check post validation", func(t *testing.T) {
		wBad := httptest.NewRecorder()
		rBad := &http.Request{
			Method: "POST",
			URL:    &url.URL{Path: "/v1/keys"},
			Body:   io.NopCloser(strings.NewReader(fmt.Sprintf(`{"ttl": %v}`, 100))),
		}
		wGood := httptest.NewRecorder()
		rGood := &http.Request{
			Method: "POST",
			URL:    &url.URL{Path: "/v1/keys"},
			Body:   io.NopCloser(strings.NewReader(fmt.Sprintf(`{"value": "%s", "ttl": %v}`, "test", 100))),
		}

		// Set up database
		db := &databaseTestImplementation{
			createCalls: []struct {
				key   string
				value string
				ttl   *int64
			}{},
			createKey:    "helloVal",
			createReturn: true,
		}
		h := NewHandler(db, slog.New(slog.DiscardHandler))
		h.ServeHTTP(wBad, rBad)
		if wBad.Code != http.StatusBadRequest {
			t.Errorf("response code = %v; want %v", wBad.Code, http.StatusBadRequest)
		}

		h.ServeHTTP(wGood, rGood)
		if wGood.Code >= 400 {
			t.Errorf("response code = %v; want response code less than 400", wGood.Code)
		}

	})
}

func TestJsonValidationPut(t *testing.T) {
	t.Run("Check post validation", func(t *testing.T) {
		wBad := httptest.NewRecorder()
		rBad := &http.Request{
			Method: "PUT",
			URL:    &url.URL{Path: fmt.Sprintf("/v1/keys/%s", "test")},
			Body:   io.NopCloser(strings.NewReader(fmt.Sprintf(`{"ttl": %v}`, 100))),
		}
		wGood := httptest.NewRecorder()
		rGood := &http.Request{
			Method: "PUT",
			URL:    &url.URL{Path: fmt.Sprintf("/v1/keys/%s", "test")},
			Body:   io.NopCloser(strings.NewReader(fmt.Sprintf(`{"value": "%s", "ttl": %v}`, "testVal", 100))),
		}

		// Set up database
		db := &databaseTestImplementation{
			createCalls: []struct {
				key   string
				value string
				ttl   *int64
			}{},
			createKey:    "helloVal",
			createReturn: true,
		}
		h := NewHandler(db, slog.New(slog.DiscardHandler))
		h.ServeHTTP(wBad, rBad)
		if wBad.Code != http.StatusBadRequest {
			t.Errorf("response code = %v; want %v", wBad.Code, http.StatusBadRequest)
		}

		h.ServeHTTP(wGood, rGood)
		if wGood.Code >= 400 {
			t.Errorf("response code = %v; want response code less than 400", wGood.Code)
		}

	})
}

package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

// DatabaseTestImplementation is an implementation of Database used for test cases
type databaseTestImplementation struct {
	createCalls []struct {
		key   string
		value string
	}
	createReturn bool
	readCalls    []struct {
		key string
	}
	readReturn  bool
	readString  string
	updateCalls []struct {
		key   string
		value string
	}
	updateReturn bool
	deleteCalls  []struct {
		key string
	}
	deleteReturn bool
}

func (db *databaseTestImplementation) Create(key string, value string) bool {
	db.createCalls = append(db.createCalls, struct {
		key   string
		value string
	}{key, value})
	return db.createReturn
}

func (db *databaseTestImplementation) Read(key string) (string, bool) {
	db.readCalls = append(db.readCalls, struct {
		key string
	}{key})
	return db.readString, db.readReturn
}

func (db *databaseTestImplementation) Update(key string, value string) bool {
	db.updateCalls = append(db.updateCalls, struct {
		key   string
		value string
	}{key, value})
	return db.updateReturn
}

func (db *databaseTestImplementation) Delete(key string) bool {
	db.deleteCalls = append(db.deleteCalls, struct {
		key string
	}{key})
	return db.deleteReturn
}

func TestWrapper_createHandler(t *testing.T) {
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name       string
		db         *databaseTestImplementation
		key        string
		value      string
		status     int
		checkCalls bool
		args       args
	}{
		{
			name: "Try to create non-existing key value pair",
			db: &databaseTestImplementation{
				createCalls: []struct {
					key   string
					value string
				}{},
				createReturn: true,
			},
			key:        "testKey",
			value:      "testValue",
			status:     http.StatusCreated,
			checkCalls: true,
			args: args{
				w: httptest.NewRecorder(),
				r: &http.Request{
					Method: "POST",
					URL:    &url.URL{Path: "/v1/key"},
				},
			},
		},
		{
			name: "Try to create an existing key value pair",
			db: &databaseTestImplementation{
				createCalls: []struct {
					key   string
					value string
				}{},
				createReturn: false,
			},
			key:        "testKey",
			value:      "testValue",
			status:     http.StatusBadRequest,
			checkCalls: true,
			args: args{
				w: httptest.NewRecorder(),
				r: &http.Request{
					Method: "POST",
					URL:    &url.URL{Path: "/v1/key"},
				},
			},
		},
		{
			name:       "Send a bad request body",
			db:         &databaseTestImplementation{},
			key:        "testKey",
			value:      `{"test": "test"}`,
			status:     http.StatusBadRequest,
			checkCalls: false,
			args: args{
				w: httptest.NewRecorder(),
				r: &http.Request{
					Method: "POST",
					URL:    &url.URL{Path: "/v1/key"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.args.r.Body = io.NopCloser(strings.NewReader(fmt.Sprintf(`{"key": "%s", "value": "%s"}`, tt.key, tt.value)))
			h := NewHandler(tt.db)
			h.ServeHTTP(tt.args.w, tt.args.r)

			resp := tt.args.w.(*httptest.ResponseRecorder)
			if resp.Code != tt.status {
				t.Errorf("Response code = %v; want %v", resp.Code, tt.status)
			}

			if tt.checkCalls {
				if len(tt.db.createCalls) == 0 {
					t.Errorf("Create() calls not created")
				}

				if tt.db.createCalls[0].key != tt.key {
					t.Errorf("Create() key = %v; want %v", tt.db.createCalls[0].key, tt.key)
				}

				if tt.db.createCalls[0].value != tt.value {
					t.Errorf("Create() value = %v; want %v", tt.db.createCalls[0].value, tt.value)
				}
			}
		})
	}
}

func TestWrapper_readHandler(t *testing.T) {
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name       string
		db         *databaseTestImplementation
		key        string
		value      string
		status     int
		checkCalls bool
		args       args
	}{
		{
			name: "Read an existing key value pair",
			db: &databaseTestImplementation{
				readCalls: []struct {
					key string
				}{},
				readReturn: true,
				readString: "testValue",
			},
			key:        "testKey",
			value:      "testValue",
			status:     http.StatusOK,
			checkCalls: true,
			args: args{
				w: httptest.NewRecorder(),
				r: &http.Request{
					Method: "GET",
					URL:    &url.URL{Path: "/v1/key"},
				},
			},
		},
		{
			name: "Try to read a non-existing key value pair",
			db: &databaseTestImplementation{
				readCalls: []struct {
					key string
				}{},
				readReturn: false,
				readString: "",
			},
			key:        "testKey",
			value:      "",
			status:     http.StatusNotFound,
			checkCalls: true,
			args: args{
				w: httptest.NewRecorder(),
				r: &http.Request{
					Method: "GET",
					URL:    &url.URL{Path: "/v1/key"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := tt.args.r.URL.Query()
			q.Set("key", tt.key)
			tt.args.r.URL.RawQuery = q.Encode()
			h := NewHandler(tt.db)
			h.ServeHTTP(tt.args.w, tt.args.r)

			resp := tt.args.w.(*httptest.ResponseRecorder)
			if resp.Code != tt.status {
				t.Errorf("Response code = %v; want %v", resp.Code, tt.status)
			}

			var body Response
			err := json.NewDecoder(resp.Body).Decode(&body)
			if err != nil {
				t.Errorf("Failed to decode response body JSON: %v", err)
			}

			expected := Response{Key: tt.key, Value: tt.value}

			if !reflect.DeepEqual(expected, body) {
				t.Errorf("Response body = %v; want %v", body, expected)
			}

			if tt.checkCalls {
				if len(tt.db.readCalls) == 0 {
					t.Errorf("Delete() calls not created")
				}

				if tt.db.readCalls[0].key != tt.key {
					t.Errorf("Delete() key = %v; want %v", tt.db.readCalls[0].key, tt.key)
				}
			}
		})
	}
}

func TestWrapper_updateHandler(t *testing.T) {
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name       string
		db         *databaseTestImplementation
		key        string
		value      string
		status     int
		checkCalls bool
		args       args
	}{
		{
			name: "Update non-existing key value pair",
			db: &databaseTestImplementation{
				updateCalls: []struct {
					key   string
					value string
				}{},
				updateReturn: false,
			},
			key:        "testKey",
			value:      "testValue",
			status:     http.StatusCreated,
			checkCalls: true,
			args: args{
				w: httptest.NewRecorder(),
				r: &http.Request{
					Method: "PUT",
					URL:    &url.URL{Path: "/v1/key"},
				},
			},
		},
		{
			name: "Update an existing key value pair",
			db: &databaseTestImplementation{
				updateCalls: []struct {
					key   string
					value string
				}{},
				updateReturn: true,
			},
			key:        "testKey",
			value:      "testValue",
			status:     http.StatusOK,
			checkCalls: true,
			args: args{
				w: httptest.NewRecorder(),
				r: &http.Request{
					Method: "PUT",
					URL:    &url.URL{Path: "/v1/key"},
				},
			},
		},
		{
			name:       "Send a bad request body",
			db:         &databaseTestImplementation{},
			key:        "testKey",
			value:      `{"test": "test"}`,
			status:     http.StatusBadRequest,
			checkCalls: false,
			args: args{
				w: httptest.NewRecorder(),
				r: &http.Request{
					Method: "PUT",
					URL:    &url.URL{Path: "/v1/key"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.args.r.Body = io.NopCloser(strings.NewReader(fmt.Sprintf(`{"key": "%s", "value": "%s"}`, tt.key, tt.value)))
			h := NewHandler(tt.db)
			h.ServeHTTP(tt.args.w, tt.args.r)

			resp := tt.args.w.(*httptest.ResponseRecorder)
			if resp.Code != tt.status {
				t.Errorf("Response code = %v; want %v", resp.Code, tt.status)
			}

			if tt.checkCalls {
				if len(tt.db.updateCalls) == 0 {
					t.Errorf("Update() calls not created")
				}

				if tt.db.updateCalls[0].key != tt.key {
					t.Errorf("Update() key = %v; want %v", tt.db.updateCalls[0].key, tt.key)
				}

				if tt.db.updateCalls[0].value != tt.value {
					t.Errorf("Update() value = %v; want %v", tt.db.updateCalls[0].value, tt.value)
				}
			}
		})
	}
}

func TestWrapper_deleteHandler(t *testing.T) {
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name       string
		db         *databaseTestImplementation
		key        string
		status     int
		checkCalls bool
		args       args
	}{
		{
			name: "Delete an existing key value pair",
			db: &databaseTestImplementation{
				deleteCalls: []struct {
					key string
				}{},
				deleteReturn: true,
			},
			key:        "testKey",
			status:     http.StatusOK,
			checkCalls: true,
			args: args{
				w: httptest.NewRecorder(),
				r: &http.Request{
					Method: "DELETE",
					URL:    &url.URL{Path: "/v1/key"},
				},
			},
		},
		{
			name: "Try to delete a non-existing key value pair",
			db: &databaseTestImplementation{
				deleteCalls: []struct {
					key string
				}{},
				deleteReturn: false,
			},
			key:        "testKey",
			status:     http.StatusNotFound,
			checkCalls: true,
			args: args{
				w: httptest.NewRecorder(),
				r: &http.Request{
					Method: "DELETE",
					URL:    &url.URL{Path: "/v1/key"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := tt.args.r.URL.Query()
			q.Set("key", tt.key)
			tt.args.r.URL.RawQuery = q.Encode()
			h := NewHandler(tt.db)
			h.ServeHTTP(tt.args.w, tt.args.r)

			resp := tt.args.w.(*httptest.ResponseRecorder)
			if resp.Code != tt.status {
				t.Errorf("Response code = %v; want %v", resp.Code, tt.status)
			}

			if tt.checkCalls {
				if len(tt.db.deleteCalls) == 0 {
					t.Errorf("Delete() calls not created")
				}

				if tt.db.deleteCalls[0].key != tt.key {
					t.Errorf("Delete() key = %v; want %v", tt.db.deleteCalls[0].key, tt.key)
				}
			}
		})
	}
}

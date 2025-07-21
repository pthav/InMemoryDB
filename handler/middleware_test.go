package handler

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/mux"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

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

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
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

func TestPrometheusMiddleware(t *testing.T) {
	requests := []struct {
		method string
		cancel bool
	}{
		{
			method: "PUT",
		},
		{
			method: "PUT",
		},
		{
			method: "GET",
		},
		{
			method: "SUB",
			cancel: true,
		},
		{
			method: "SUB",
			cancel: true,
		},
		{
			method: "SUB",
			cancel: false,
		},
		{
			method: "SUB",
			cancel: false,
		},
		{
			method: "PUB",
		},
		{
			method: "PUB",
		},
		{
			method: "PUB",
		},
	}

	t.Run("Testing metrics", func(t *testing.T) {
		db := &databaseTestImplementation{
			mu:         sync.RWMutex{},
			readReturn: true,
			putReturn:  true,
		}
		discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
		h := NewHandler(db, discardLogger)
		s := httptest.NewServer(h)

		// Send all requests
		var wg sync.WaitGroup
		for _, r := range requests {
			wg.Add(1)
			go func() {
				defer wg.Done()
				client := http.Client{}

				switch r.method {
				case "PUT":
					req, err := http.NewRequest("PUT", s.URL+"/v1/keys/test", io.NopCloser(strings.NewReader(`{"key":"test","value":"test"}`)))
					if err != nil {
						t.Errorf("Error creating request: %v", err)
					}
					_, err = client.Do(req)
					if err != nil {
						t.Errorf("Error creating request: %v", err)
					}

				case "GET":
					req, err := http.NewRequest("GET", s.URL+"/v1/keys/test", nil)
					if err != nil {
						t.Errorf("Error creating request: %v", err)
					}
					_, err = client.Do(req)
					if err != nil {
						t.Errorf("Error creating request: %v", err)
					}

				case "PUB":
					req, err := http.NewRequest("POST", s.URL+"/v1/publish/channel", io.NopCloser(strings.NewReader(`{"message":"m"}`)))
					if err != nil {
						t.Errorf("Error creating request: %v", err)
					}
					_, err = client.Do(req)
					if err != nil {
						t.Errorf("Error creating request: %v", err)
					}

				case "SUB":
					if r.cancel {
						ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
						defer cancel()
						req, err := http.NewRequestWithContext(ctx, "GET", s.URL+"/v1/subscribe/channel", nil)
						if err != nil {
							t.Errorf("Error creating new request: %v", err)
						}

						_, _ = client.Do(req)
					} else {
						go func() {
							_, _ = client.Get(s.URL + "/v1/subscribe/channel")
						}()
						<-time.After(100 * time.Millisecond)
						wg.Add(1)
						wg.Done()
					}
				}
			}()
		}

		wg.Wait()

		t.Log(s.URL)

		// Check metrics
		getMetric := testutil.ToFloat64(h.m.dbHttpRequestCounter.WithLabelValues("GET", "/v1/keys/", "200"))
		if getMetric != 1 {
			t.Errorf("Metric does not match: got %v, want %v", getMetric, 1)
		}

		putMetric := testutil.ToFloat64(h.m.dbHttpRequestCounter.WithLabelValues("PUT", "/v1/keys/", "200"))
		if putMetric != 2 {
			t.Errorf("Metric does not match: got %v, want %v", putMetric, 1)
		}

		publishedMessages := testutil.ToFloat64(h.m.dbPublishedMessages)
		if publishedMessages != 3 {
			t.Errorf("Expected %v published messages but got %v", 3, publishedMessages)
		}

		subscribers := testutil.ToFloat64(h.m.dbSubscriptions)
		if subscribers != 2 {
			t.Errorf("Expected %v subscriptions but got %v", 2, subscribers)
		}
	})
}

package endpoint

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type publishRequest struct {
	Message string `json:"message" validate:"required"`
}

type testHandler struct {
	router   *mux.Router
	channels map[string][]chan string
	mu       sync.RWMutex
}

func newTestHandler() *testHandler {
	h := &testHandler{
		channels: map[string][]chan string{
			"test": make([]chan string, 2),
			"dogs": make([]chan string, 1)},
		mu: sync.RWMutex{},
	}

	r := mux.NewRouter()
	r.HandleFunc("/v1/subscribe/{channel}", h.subscribe).Methods("GET")
	r.HandleFunc("/v1/publish/{channel}", h.publish).Methods("POST")
	h.router = r
	return h
}

func (h *testHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

func (h *testHandler) subscribe(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	channel := vars["channel"]

	// Check if SSE is valid for the writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
	}

	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	c := make(chan string, 10)

	// Run a go func to remove the subscriber from the channel when they disconnect
	ctx := r.Context()
	go func() {
		<-ctx.Done()
		h.mu.Lock()
		for i, ch := range h.channels[channel] {
			if ch == c {
				h.channels[channel] = append(h.channels[channel][:i], h.channels[channel][i+1:]...)
				break
			}
		}
		close(c)
		h.mu.Unlock()
	}()

	h.mu.Lock()
	h.channels[channel] = append(h.channels[channel], c)
	h.mu.Unlock()

	for message := range c {
		_, err := fmt.Fprintf(w, "data: %s\n\n", message)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		flusher.Flush()
	}
}

func (h *testHandler) publish(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	channel := vars["channel"]

	var pData publishRequest
	if err := json.NewDecoder(r.Body).Decode(&pData); err != nil {
		http.Error(w, "Publish request has bad body", http.StatusBadRequest)
	}

	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte(`{"error":"null"}"`))
	if err != nil {
		return
	}

	h.mu.RLock()
	for _, c := range h.channels[channel] {
		select {
		case c <- pData.Message:
		default:
			// Drop message if the channel is full
		}
	}
	h.mu.RUnlock()
}

func TestCommand_pubSub(t *testing.T) {
	type subscriber struct {
		channel  string
		expected []string
		expire   string
	}

	type publisher struct {
		channel string
		message string
		wait    time.Duration
	}

	tests := []struct {
		t           testCase
		subscribers []subscriber
		publishers  []publisher
	}{
		{
			t: testCase{
				name: "One subscriber",
			},
			subscribers: []subscriber{
				{channel: "test", expected: []string{"message1", "message2"}, expire: "1"},
			},
			publishers: []publisher{
				{channel: "test", message: "message1", wait: 10 * time.Millisecond},
				{channel: "test", message: "message2", wait: 20 * time.Millisecond},
			},
		},
		{
			t: testCase{
				name: "Multiple subscribers",
			},
			subscribers: []subscriber{
				{channel: "test", expected: []string{"message1", "message2"}, expire: "1"},
				{channel: "test", expected: []string{"message1", "message2"}, expire: "1"},
				{channel: "dogs", expected: []string{"message1", "message2", "message3", "message4"}, expire: "1"},
			},
			publishers: []publisher{
				{channel: "test", message: "message1", wait: 10 * time.Millisecond},
				{channel: "test", message: "message2", wait: 20 * time.Millisecond},
				{channel: "dogs", message: "message1", wait: 10 * time.Millisecond},
				{channel: "dogs", message: "message2", wait: 20 * time.Millisecond},
				{channel: "dogs", message: "message3", wait: 30 * time.Millisecond},
				{channel: "dogs", message: "message4", wait: 40 * time.Millisecond},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.t.name, func(t *testing.T) {
			ts := httptest.NewServer(newTestHandler())
			defer ts.Close()

			// Start each subscriber
			var wg sync.WaitGroup
			for i, s := range tt.subscribers {
				wg.Add(1)
				go func() {
					defer wg.Done()
					t.Logf("Subscriber %v subscribing to channel %v", i, s.channel)

					args := []string{"subscribe", "-c", s.channel, "-t", s.expire, "-u", ts.URL}
					output, err := execute(t, NewEndpointsCmd(), args...)
					if err != nil {
						t.Error(err)
						return
					}

					// Get each message
					messageCount := 0
					scanner := bufio.NewScanner(strings.NewReader(output))
					for scanner.Scan() {
						line := scanner.Text()
						t.Logf("Subscriber %v has received line %v", i, line)

						// Only check valid SSE output
						if strings.HasPrefix(line, "data: ") {
							if messageCount > len(s.expected) {
								t.Errorf("Too many messages received got %v expected %v", messageCount, len(s.expected))
								break
							}
							msg := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
							if msg != s.expected[messageCount] {
								t.Errorf("For message %v expected %v but got %v", messageCount, s.expected[messageCount], msg)
								break
							}
							messageCount++
						}
					}
					if messageCount != len(s.expected) {
						t.Errorf("Incorrect message count got %v expected %v", messageCount, len(s.expected))
					}
					if err = scanner.Err(); err != nil {
						t.Error(err)
					}
				}()
			}

			// Start each publisher
			for _, p := range tt.publishers {
				wg.Add(1)
				go func() {
					defer wg.Done()
					args := []string{"publish", "-c", p.channel, "-m", p.message, "-u", ts.URL}
					<-time.After(p.wait)
					_, err := execute(t, NewEndpointsCmd(), args...)
					if err != nil {
						t.Errorf("Error executing publish: %v", err)
					}
				}()
			}

			wg.Wait()
		})
	}
}

func TestCommand_pubSubValidation(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "Test subscribe errors without channel",
			args: []string{"subscribe"},
		},
		{
			name: "Test publish errors without channel",
			args: []string{"publish", "-m", "message"},
		},
		{
			name: "Test publish errors without message",
			args: []string{"publish", "-c", "channel"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := execute(t, NewEndpointsCmd(), tt.args...)
			if err == nil {
				t.Error("Expected err but got nil")
			} else if !strings.Contains(err.Error(), "required") {
				t.Errorf("Expected error to contain %v, got %v", "required", err)
			}
		})
	}
}

package handler

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestWrapper_pubSub(t *testing.T) {
	type subscriber struct {
		channel  string
		expected []string
		expire   time.Duration
	}

	type publisher struct {
		channel string
		message string
		wait    time.Duration
	}

	tests := []struct {
		name        string
		subscribers []subscriber
		publishers  []publisher
		wait        time.Duration
	}{
		{
			name: "One subscriber",
			subscribers: []subscriber{
				{channel: "test", expected: []string{"message1", "message2"}, expire: time.Second},
			},
			publishers: []publisher{
				{channel: "test", message: "message1", wait: 10 * time.Millisecond},
				{channel: "test", message: "message2", wait: 20 * time.Millisecond},
			},
			wait: 100 * time.Millisecond,
		},
		{
			name: "Multiple subscribers",
			subscribers: []subscriber{
				{channel: "test", expected: []string{"message1", "message2"}, expire: time.Second},
				{channel: "test", expected: []string{"message1", "message2"}, expire: time.Second},
				{channel: "dogs", expected: []string{"message1", "message2", "message3", "message4"}, expire: time.Second},
			},
			publishers: []publisher{
				{channel: "test", message: "message1", wait: 10 * time.Millisecond},
				{channel: "test", message: "message2", wait: 20 * time.Millisecond},
				{channel: "dogs", message: "message1", wait: 10 * time.Millisecond},
				{channel: "dogs", message: "message2", wait: 20 * time.Millisecond},
				{channel: "dogs", message: "message3", wait: 30 * time.Millisecond},
				{channel: "dogs", message: "message4", wait: 40 * time.Millisecond},
			},
			wait: 100 * time.Millisecond,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up handler
			db := &databaseTestImplementation{}
			h := NewHandler(db, slog.New(slog.DiscardHandler))
			ts := httptest.NewServer(h)
			defer ts.Close()

			// Start each subscriber
			for i, s := range tt.subscribers {
				go func() {
					t.Logf("Subscriber %v subscribing to channel %v", i, s.channel)

					// Create an http request for subscription that will automatically disconnect after the
					// subscriber's expiration
					client := http.Client{}

					ctx, cancel := context.WithTimeout(context.Background(), s.expire)
					defer cancel()

					req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/v1/subscribe/%s", ts.URL, s.channel), nil)
					if err != nil {
						t.Error(err)
					}

					resp, err := client.Do(req)
					if err != nil {
						t.Error(err)
					}

					defer resp.Body.Close()
					reader := bufio.NewReader(resp.Body)

					// Get each message
					messageCount := 0
					for {
						line, err := reader.ReadString('\n')
						if err != nil {
							// If it is an organic error, check the final messageCount against the expectation
							if errors.Is(err, context.DeadlineExceeded) || err == io.EOF {
								if messageCount != len(s.expected) {
									t.Errorf("Got message count %v expected %v", messageCount, len(s.expected))
								}
								break
							}
							t.Errorf("Message read error: %v", err)
							break
						}
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
				}()
			}

			// Start each publisher
			for _, p := range tt.publishers {
				go func() {
					<-time.After(p.wait)
					t.Logf("Publishing to channel %v with message %v", p.channel, p.message)
					payload := fmt.Sprintf(`{"message": "%v"}`, p.message)
					resp, err := http.Post(fmt.Sprintf("%s/v1/publish/%s", ts.URL, p.channel), "application/json", strings.NewReader(payload))
					defer func(Body io.ReadCloser) {
						err := Body.Close()
						if err != nil {
							t.Errorf("Failed to close response body: %v", err)
						}
					}(resp.Body)
					if err != nil {
						t.Errorf("Unable to send post request: %v", err)
					}
				}()
			}

			<-time.After(tt.wait)
		})
	}
}

func TestJsonValidationPost(t *testing.T) {
	t.Run("Check post validation", func(t *testing.T) {
		// Don't pass in a value
		wBad := httptest.NewRecorder()
		rBad := &http.Request{
			Method: "POST",
			URL:    &url.URL{Path: "/v1/keys"},
			Body:   io.NopCloser(strings.NewReader(fmt.Sprintf(`{"ttl": %v}`, 100))),
		}

		// Pass in value as required
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
		// Don't pass in a value
		wBad := httptest.NewRecorder()
		rBad := &http.Request{
			Method: "PUT",
			URL:    &url.URL{Path: fmt.Sprintf("/v1/keys/%s", "test")},
			Body:   io.NopCloser(strings.NewReader(fmt.Sprintf(`{"ttl": %v}`, 100))),
		}

		// Pass in value as required
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

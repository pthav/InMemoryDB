package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Status wrapper so that the middleware can access information after calling next()
type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
	e          string
}

// Flush is necessary here for the subscribe functionality to work
func (w *statusResponseWriter) Flush() {
	w.ResponseWriter.(http.Flusher).Flush()
}

// WriteHeader enables the collection of status codes
func (w *statusResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// loggingMiddleware logs all incoming requests
func (h *Wrapper) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get body data
		if r.Body != nil && r.ContentLength != 0 {
			var rData map[string]any
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// Unmarshal request body
			if err = json.Unmarshal(bodyBytes, &rData); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			} else {
				// Get body data to request
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				h.logger.Info(
					"incoming request",
					"method", r.Method,
					"URI", r.RequestURI,
					"Body", rData)
			}
		} else {
			h.logger.Info(
				"incoming request",
				"method", r.Method,
				"URI", r.RequestURI)
		}

		sw, ok := w.(*statusResponseWriter)
		if !ok {
			sw = &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		}
		next.ServeHTTP(sw, r)

		if sw.statusCode >= 400 {
			h.logger.Error("request failed", "method", r.Method, "URI", r.RequestURI, "err", sw.e)
		}
	})
}

// prometheusMiddleware handles all prometheus metric updates.
func (h *Wrapper) prometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		var url string
		rawURL := r.URL.String()
		switch {
		case strings.Contains(rawURL, "publish"):
			url = "/v1/publish/"
		case strings.Contains(rawURL, "subscribe"):
			url = "/v1/subscribe/"
		case strings.Contains(rawURL, "ttl"):
			url = "/v1/ttl/"
		case rawURL == "/v1/keys":
			url = "/v1/keys"
		default:
			url = "/v1/keys/"
		}

		// Subscription gauge
		if strings.Contains(r.URL.Path, "subscribe") {
			h.m.dbSubscriptions.Inc()
		}

		before := time.Now().UnixMilli()
		next.ServeHTTP(sw, r)
		after := time.Now().UnixMilli()

		// Observe metrics

		// Request counter
		requestCounter, err := h.m.dbHttpRequestCounter.GetMetricWithLabelValues(
			r.Method,
			url,
			fmt.Sprintf("%v", sw.statusCode),
		)

		if err == nil {
			requestCounter.Inc()
		} else {
			h.logger.Error("prometheus metrics error", "err", err)
		}

		// Latency histogram
		latency, err := h.m.dbLatency.GetMetricWithLabelValues(
			r.Method,
			url,
			fmt.Sprintf("%v", sw.statusCode),
		)

		if err == nil {
			l := float64(after - before)
			latency.Observe(l)
		} else {
			h.logger.Error("prometheus metrics error", "err", err)
		}

		// Published messages counter
		if strings.Contains(r.URL.Path, "publish") && sw.statusCode < 300 {
			h.m.dbPublishedMessages.Inc()
		}

		// Subscription gauge
		if strings.Contains(r.URL.Path, "subscribe") {
			h.m.dbSubscriptions.Dec()
		}
	})
}

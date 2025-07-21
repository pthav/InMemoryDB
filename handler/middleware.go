package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

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
		next.ServeHTTP(w, r)
	})
}

// prometheusMiddleware handles all prometheus metric updates.
func (h *Wrapper) prometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

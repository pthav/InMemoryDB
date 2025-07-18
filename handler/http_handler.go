package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
	"io"
	"log/slog"
	"net/http"
	"sync"
)

type database interface {
	Create(data struct {
		Value string `json:"value"`
		Ttl   *int64 `json:"ttl"`
	}) (bool, string) // Create a UUID for the value and add it if it doesn't exist
	Get(key string) (string, bool) // Get the associated value if it exists and hasn't expired
	Put(data struct {
		Key   string `json:"key"`
		Value string `json:"value"`
		Ttl   *int64 `json:"ttl"`
	}) bool // Put a key, value pair
	Delete(key string) bool           // Delete the key, value pair
	GetTTL(key string) (*int64, bool) // Get the remaining TTL for a given key if it has a TTL
}

type postResponse struct {
	Key string `json:"key"`
}

type getResponse struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type getTTLResponse struct {
	Key string `json:"key"`
	TTL *int64 `json:"ttl"`
}

type postRequest struct {
	Value string `json:"value" validate:"required"`
	Ttl   *int64 `json:"ttl"`
}

type putRequest struct {
	Key   string `json:"key"` // This is overwritten by the url parameter if passed in with the request body
	Value string `json:"value" validate:"required"`
	Ttl   *int64 `json:"ttl"`
}

type pubSubBroker struct {
	mu       sync.RWMutex
	channels map[string][]chan string
}

type publishRequest struct {
	Message string `json:"message" validate:"required"`
}

type Wrapper struct {
	db     database
	router *mux.Router
	logger *slog.Logger
	broker pubSubBroker
}

// Helper function for writing JSON errors
func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(map[string]string{
		"error": msg,
	})
	if err != nil {
		return
	}
}

// NewHandler Return a new HandlerWrapper instance with all routes set
func NewHandler(db database, logger *slog.Logger) *Wrapper {
	handler := &Wrapper{db: db, logger: logger, broker: pubSubBroker{channels: make(map[string][]chan string)}}
	handler.router = mux.NewRouter()
	handler.router.HandleFunc("/v1/keys", handler.postHandler).
		Methods("POST")
	handler.router.HandleFunc("/v1/keys/{key}", handler.getHandler).
		Methods("GET")
	handler.router.HandleFunc("/v1/keys/{key}", handler.putHandler).
		Methods("PUT")
	handler.router.HandleFunc("/v1/keys/{key}", handler.deleteHandler).
		Methods("DELETE")
	handler.router.HandleFunc("/v1/ttl/{key}", handler.getTTLHandler).
		Methods("GET")
	handler.router.HandleFunc("/v1/subscribe/{channel}", handler.subscribeHandler).
		Methods("GET")
	handler.router.HandleFunc("/v1/publish/{channel}", handler.publishHandler).
		Methods("POST")
	handler.router.Use(handler.loggingMiddleware)
	return handler
}

func (h *Wrapper) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	h.router.ServeHTTP(writer, request)
}

// postHandler uses request key and value from the request body to set the key value pair in the database
func (h *Wrapper) postHandler(w http.ResponseWriter, r *http.Request) {
	var rData postRequest
	err := json.NewDecoder(r.Body).Decode(&rData)
	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Validate the input
	validate := validator.New()
	err = validate.Struct(rData)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("Validation errors when parsing post request: %s", err.Error()))
		return
	}

	// Forward the post request
	set, key := h.db.Create(struct {
		Value string `json:"value"`
		Ttl   *int64 `json:"ttl"`
	}(rData))

	if !set {
		writeJSONError(w, http.StatusInternalServerError, "Failed while adding key-value pair to store")
		return
	}

	w.WriteHeader(http.StatusCreated)
	response := postResponse{Key: key}

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		h.logger.Error("Error occurred while encoding json to post request", "error: ", err)
	}
}

// getHandler uses the request key and returns the associated value if it exists
func (h *Wrapper) getHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]
	value, loaded := h.db.Get(key)
	response := getResponse{Key: key, Value: value}
	w.Header().Set("Content-Type", "application/json")

	if !loaded {
		writeJSONError(w, http.StatusNotFound, "Key not found")
		return
	}

	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
}

// putHandler uses request key and value from the request body to set the key value pair in the database
// Users are allowed to update the ttl through "PUT" operations.
func (h *Wrapper) putHandler(w http.ResponseWriter, r *http.Request) {
	var rData putRequest
	err := json.NewDecoder(r.Body).Decode(&rData)
	vars := mux.Vars(r)
	rData.Key = vars["key"]

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate the input
	validate := validator.New()
	err = validate.Struct(rData)
	if err != nil {
		http.Error(w, fmt.Sprintf("Validation errors when parsing put request: %s", err), http.StatusBadRequest)
		return
	}

	// Forward the put request
	set := h.db.Put(struct {
		Key   string `json:"key"`
		Value string `json:"value"`
		Ttl   *int64 `json:"ttl"`
	}(rData))
	if set {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
}

// deleteHandler uses the request key to delete the key value pair from the database
func (h *Wrapper) deleteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]
	deleted := h.db.Delete(key)
	if deleted {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

// getTTLHandler will get the remaining TTL for a key value pair
func (h *Wrapper) getTTLHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]
	ttl, loaded := h.db.GetTTL(key)
	response := getTTLResponse{Key: key, TTL: ttl}
	w.Header().Set("Content-Type", "application/json")

	if !loaded {
		writeJSONError(w, http.StatusNotFound, "Key not found")
		return
	}

	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
	}
}

// subscribeHandler allows a client to subscribe to a specific channel and receive string messages over the channel
func (h *Wrapper) subscribeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	channel := vars["channel"]

	// Check if SSE is valid for the writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	c := make(chan string, 10)

	h.broker.mu.Lock()
	h.broker.channels[channel] = append(h.broker.channels[channel], c)
	h.broker.mu.Unlock()

	// Run a go func to remove the subscriber from the channel when they disconnect
	ctx := r.Context()
	go func() {
		<-ctx.Done()
		h.broker.mu.Lock()
		for i, ch := range h.broker.channels[channel] {
			if ch == c {
				h.broker.channels[channel] = append(h.broker.channels[channel][:i], h.broker.channels[channel][i+1:]...)
				break
			}
		}
		close(c)
		h.broker.mu.Unlock()
	}()

	for message := range c {
		_, err := fmt.Fprintf(w, "data: %s\n\n", message)
		if err != nil {
			http.Error(w, "Error writing message to subscriber", http.StatusInternalServerError)
		}
		flusher.Flush()
	}
}

// publishHandler allows a client to publish a string message to a specific channel for all subscribers
func (h *Wrapper) publishHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	channel := vars["channel"]

	var pData publishRequest
	if err := json.NewDecoder(r.Body).Decode(&pData); err != nil {
		http.Error(w, "Publish request has bad body", http.StatusBadRequest)
	}

	h.broker.mu.RLock()
	defer h.broker.mu.RUnlock()

	for _, c := range h.broker.channels[channel] {
		select {
		case c <- pData.Message:
		default:
			// Drop message if the channel is full
		}
	}

	w.WriteHeader(http.StatusNoContent)
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
		next.ServeHTTP(w, r)
	})
}

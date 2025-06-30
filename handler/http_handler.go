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
	Delete(key string) bool          // Delete the key, value pair
	GetTTL(key string) (int64, bool) // Get the remaining TTL for a given key if it has a TTL
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
	TTL int64  `json:"ttl"`
}

type postRequest struct {
	Value string `json:"value" validate:"required"`
	Ttl   *int64 `json:"ttl"`
}

type putRequest struct {
	Key   string `json:"key" validate:"required"`
	Value string `json:"value" validate:"required"`
	Ttl   *int64 `json:"ttl"`
}

type Wrapper struct {
	db     database
	router *mux.Router
	logger *slog.Logger
}

// NewHandler Return a new HandlerWrapper instance with all routes set
func NewHandler(db database, logger *slog.Logger) *Wrapper {
	handler := &Wrapper{db: db, logger: logger}
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
	handler.router.Use(handler.loggingMiddleware)
	return handler
}

func (h *Wrapper) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	h.router.ServeHTTP(writer, request)
}

// setHandler use request key and value from the request body to set the key value pair in the database
func (h *Wrapper) postHandler(w http.ResponseWriter, r *http.Request) {
	var rData postRequest
	err := json.NewDecoder(r.Body).Decode(&rData)
	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate the input
	validate := validator.New()
	err = validate.Struct(rData)
	if err != nil {
		http.Error(w, fmt.Sprintf("Validation errors when parsing create request: %s", err.Error()), http.StatusBadRequest)
		return
	}

	// Forward the post request
	set, key := h.db.Create(struct {
		Value string `json:"value"`
		Ttl   *int64 `json:"ttl"`
	}(rData))

	if !set {
		http.Error(w, "Could not add value to store", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	response := postResponse{Key: key}

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// getHandler use request key and return associated value if it exists
func (h *Wrapper) getHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]
	value, loaded := h.db.Get(key)
	response := getResponse{Key: key, Value: value}
	w.Header().Set("Content-Type", "application/json")

	if !loaded {
		w.WriteHeader(http.StatusNotFound)
	}

	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// setHandler use request key and value from the request body to set the key value pair in the database
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
		http.Error(w, fmt.Sprintf("Validation errors when parsing update request: %s", err), http.StatusBadRequest)
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

// deleteHandler use the request key to delete the key value pair from the database
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

func (h *Wrapper) getTTLHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]
	ttl, loaded := h.db.GetTTL(key)
	response := getTTLResponse{Key: key, TTL: ttl}
	w.Header().Set("Content-Type", "application/json")

	if !loaded {
		w.WriteHeader(http.StatusNotFound)
	}

	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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

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
		Ttl   *int   `json:"ttl"`
	}) (bool, string)
	Read(key string) (string, bool)
	Update(data struct {
		Key   string `json:"key"`
		Value string `json:"value"`
		Ttl   *int   `json:"ttl"`
	}) bool
	Delete(key string) bool
}

type createResponse struct {
	Key string `json:"key"`
}

type readResponse struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type createRequest struct {
	Value string `json:"value" validate:"required"`
	Ttl   *int   `json:"ttl"`
}

type updateRequest struct {
	Key   string `json:"key" validate:"required"`
	Value string `json:"value" validate:"required"`
	Ttl   *int   `json:"ttl"`
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
	handler.router.HandleFunc("/v1/keys", handler.createHandler).
		Methods("POST")
	handler.router.HandleFunc("/v1/keys/{key}", handler.readHandler).
		Methods("GET")
	handler.router.HandleFunc("/v1/keys/{key}", handler.updateHandler).
		Methods("PUT")
	handler.router.HandleFunc("/v1/keys/{key}", handler.deleteHandler).
		Methods("DELETE")
	handler.router.Use(handler.loggingMiddleware)
	return handler
}

func (h *Wrapper) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	h.router.ServeHTTP(writer, request)
}

// setHandler use request key and value from the request body to set the key value pair in the database
func (h *Wrapper) createHandler(w http.ResponseWriter, r *http.Request) {
	var rData createRequest
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

	set, key := h.db.Create(struct {
		Value string `json:"value"`
		Ttl   *int   `json:"ttl"`
	}(rData))

	if !set {
		http.Error(w, "Could not add value to store", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	response := createResponse{Key: key}

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// getHandler use request key and return associated value if it exists
func (h *Wrapper) readHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]
	value, loaded := h.db.Read(key)
	response := readResponse{Key: key, Value: value}
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
func (h *Wrapper) updateHandler(w http.ResponseWriter, r *http.Request) {
	var rData updateRequest
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

	set := h.db.Update(struct {
		Key   string `json:"key"`
		Value string `json:"value"`
		Ttl   *int   `json:"ttl"`
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

// loggingMiddleware logs all incoming requests
func (h *Wrapper) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read body data
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
				// Read body data to request
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

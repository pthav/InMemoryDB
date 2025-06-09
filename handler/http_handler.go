package handler

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/mux"
	"io"
	"log/slog"
	"net/http"
)

type database interface {
	Create(key string, value string) bool
	Read(key string) (string, bool)
	Update(key string, value string) bool
	Delete(key string) bool
}

type response struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type requestData struct {
	Key   string `json:"key"`
	Value string `json:"value"`
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
	var rData requestData
	err := json.NewDecoder(r.Body).Decode(&rData)
	if err == nil {
		set := h.db.Create(rData.Key, rData.Value)
		if set {
			w.WriteHeader(http.StatusCreated)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	} else {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

// getHandler use request key and return associated value if it exists
func (h *Wrapper) readHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]
	value, loaded := h.db.Read(key)
	response := response{Key: key, Value: value}
	w.Header().Set("Content-Type", "application/json")

	if loaded {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}

	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// setHandler use request key and value from the request body to set the key value pair in the database
func (h *Wrapper) updateHandler(w http.ResponseWriter, r *http.Request) {
	var rData requestData
	err := json.NewDecoder(r.Body).Decode(&rData)
	vars := mux.Vars(r)
	rData.Key = vars["key"]
	if err == nil {
		set := h.db.Update(rData.Key, rData.Value)
		if set {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusCreated)
		}
	} else {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
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

func (h *Wrapper) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read body data
		if r.Body != nil && r.ContentLength != 0 {
			var rData requestData
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if err = json.Unmarshal(bodyBytes, &rData); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			} else {
				// Readd body data to request
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

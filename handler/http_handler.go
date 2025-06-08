package handler

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"
)

type Database interface {
	Create(key string, value string) bool
	Read(key string) (string, bool)
	Update(key string, value string) bool
	Delete(key string) bool
}

type Response struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Status int    `json:"status"`
}

type RequestData struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Wrapper struct {
	db     Database
	router *mux.Router
}

// NewHandler Return a new HandlerWrapper instance with all routes set
func NewHandler(db Database) *Wrapper {
	handler := &Wrapper{db: db}
	handler.router = mux.NewRouter()
	handler.router.HandleFunc("/v1/key", handler.createHandler).
		Methods("POST")
	handler.router.HandleFunc("/v1/key", handler.readHandler).
		Methods("GET")
	handler.router.HandleFunc("/v1/key", handler.updateHandler).
		Methods("PUT")
	handler.router.HandleFunc("/v1/key", handler.deleteHandler).
		Methods("DELETE")
	return handler
}

func (h *Wrapper) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	h.router.ServeHTTP(writer, request)
}

// setHandler use request key and value from the request body to set the key value pair in the database
func (h *Wrapper) createHandler(w http.ResponseWriter, r *http.Request) {
	var requestData RequestData
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err == nil {
		set := h.db.Create(requestData.Key, requestData.Value)
		if set {
			w.WriteHeader(http.StatusCreated)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	} else {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

// getHandler use request key and return associated value if it exists
func (h *Wrapper) readHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	value, loaded := h.db.Read(key)
	response := Response{Key: key, Value: value}
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
	var requestData RequestData
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err == nil {
		set := h.db.Update(requestData.Key, requestData.Value)
		if set {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusCreated)
		}
	} else {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

// deleteHandler use the request key to delete the key value pair from the database
func (h *Wrapper) deleteHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	deleted := h.db.Delete(key)
	if deleted {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

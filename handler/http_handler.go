package handler

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"
)

type Database interface {
	Get(key string) (string, bool)
	Set(key string, value string) bool
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
	handler.router.HandleFunc("/v1/get", handler.getHandler).
		Methods("GET")
	handler.router.HandleFunc("/v1/set", handler.setHandler).
		Methods("POST")
	handler.router.HandleFunc("/v1/delete", handler.deleteHandler).
		Methods("DELETE")
	return handler
}

func (h *Wrapper) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	h.router.ServeHTTP(writer, request)
}

// getHandler use request key and return associated value if it exists
func (h *Wrapper) getHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	value, loaded := h.db.Get(key)
	if loaded {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		response := Response{Key: key, Value: value, Status: http.StatusOK}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

// setHandler use request key and value from the request body to set the key value pair in the database
func (h *Wrapper) setHandler(w http.ResponseWriter, r *http.Request) {
	var requestData RequestData
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err == nil {
		set := h.db.Set(requestData.Key, requestData.Value)
		if set {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusBadRequest)
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

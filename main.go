package main

import (
	"InMemoryDB/database"
	"InMemoryDB/handler"
	"log/slog"
	"net/http"
	"os"
)

// CLEANER GO ROUTINE FOR HEAP

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	h := handler.NewHandler(database.NewInMemoryDatabase(), logger)
	err := http.ListenAndServe("localhost:8080", h)
	if err != nil {
		return
	}
}

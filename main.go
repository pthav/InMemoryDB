package main

import (
	"InMemoryDB/database"
	"InMemoryDB/handler"
	"log/slog"
	"net/http"
	"os"
)

// Finish persistence (test loading). Start PUB/SUB

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	db := database.NewInMemoryDatabase(database.WithInitialData("startup.json"), database.WithLogger(logger))
	h := handler.NewHandler(db, logger)
	err := http.ListenAndServe("localhost:8080", h)
	if err != nil {
		return
	}
}

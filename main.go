package main

import (
	"InMemoryDB/database"
	"InMemoryDB/handler"
	"net/http"
)

func main() {
	h := handler.NewHandler(database.NewInMemoryDatabase())
	err := http.ListenAndServe("localhost:8080", h)
	if err != nil {
		return
	}
}

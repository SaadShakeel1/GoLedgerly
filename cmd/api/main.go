package main

import (
	"log"
	"net/http"

	"github.com/SaadShakeel1/goledgerly/internal/store"
)

type application struct {
	store *store.Store
}

func main() {
	app := &application{
		store: store.New(),
	}

	mux := app.routes()

	log.Println("starting server on :4000")
	err := http.ListenAndServe(":4000", mux)
	log.Fatal(err)
}

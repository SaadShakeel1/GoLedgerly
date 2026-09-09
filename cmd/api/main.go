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

	s, err := store.New(1000, "./store.wal")
	if err != nil {
		log.Fatal(err)
	}
	app := &application{
		store: s,
	}

	mux := app.routes()

	log.Println("starting server on :4000")
	errr := http.ListenAndServe(":4000", mux)
	log.Fatal(errr)
}

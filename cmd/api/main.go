package main

import (
	"log"
	"net/http"
	"os"

	"github.com/SaadShakeel1/goledgerly/internal/store"
)

type application struct {
	store     *store.Store
	apiKey    string
	jwtSecret []byte
}

func main() {

	s, err := store.New(1000, "./store.wal")
	if err != nil {
		log.Fatal(err)
	}
	apiKey := os.Getenv("LEDGERLY_API_KEY")
	if apiKey == "" {
		log.Fatal("LEDGERLY_API_KEY must be set")
	}
	jwtSecret := os.Getenv("LEDGERLY_JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("LEDGERLY_JWT_SECRET must be set")
	}
	app := &application{
		store:     s,
		apiKey:    apiKey,
		jwtSecret: []byte(jwtSecret),
	}

	mux := app.routes()

	log.Println("starting server on :4000")
	errr := http.ListenAndServe(":4000", mux)
	log.Fatal(errr)
}

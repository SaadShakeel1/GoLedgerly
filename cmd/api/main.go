package main

import (
	"io"
	"log"
	"net/http"

	"github.com/SaadShakeel1/goledgerly/internal/store"
)

func main() {
	s := store.New()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("GET /store/{key}", func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")
		value, found := s.Get(key)
		if !found {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(value))
	})
	mux.HandleFunc("PUT /store/{key}", func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		s.Set(key, string(body))
		w.WriteHeader(http.StatusCreated)
	})

	mux.HandleFunc("DELETE /store/{key}", func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")
		s.Delete(key)
		w.WriteHeader(http.StatusNoContent)
	})
	log.Println("starting server on : 4000")
	err := http.ListenAndServe(":4000", mux)
	log.Fatal(err)
}

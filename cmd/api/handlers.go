package main

import (
	"io"
	"net/http"
)

func (app *application) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (app *application) getHandler(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	value, found := app.store.Get(key)
	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(value))
}

func (app *application) putHandler(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	app.store.Set(key, string(body))
	w.WriteHeader(http.StatusCreated)
}

func (app *application) deleteHandler(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	app.store.Delete(key)
	w.WriteHeader(http.StatusNoContent)
}

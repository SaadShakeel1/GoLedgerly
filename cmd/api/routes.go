package main

import "net/http"

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", app.healthHandler)
	mux.HandleFunc("GET /store/{key}", app.getHandler)
	mux.HandleFunc("PUT /store/{key}", app.putHandler)
	mux.HandleFunc("DELETE /store/{key}", app.deleteHandler)
	return mux
}

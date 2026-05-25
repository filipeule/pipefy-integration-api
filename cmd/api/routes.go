package main

import "net/http"

func (app *application) mount() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /clientes", app.createCardHandler)
	mux.HandleFunc("POST /webhooks/pipefy/card-updated", app.updateCardHandler)

	return app.recoverMiddleware(mux)
}

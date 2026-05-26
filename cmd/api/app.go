package main

import (
	"encoding/json"
	"net/http"
	"pipefy-integration/internal/service"
	"pipefy-integration/internal/validate"
)

type application struct {
	port            string
	validator       *validate.Validator
	databaseService *service.DatabaseService
	pipefyService   *service.PipefyService
}

func (app *application) readJSON(w http.ResponseWriter, r *http.Request, data any) error {
	maxBytes := 1_048_578
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	return decoder.Decode(data)
}

func (app *application) writeJSON(w http.ResponseWriter, status int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if data == nil {
		return nil
	}

	return json.NewEncoder(w).Encode(data)
}

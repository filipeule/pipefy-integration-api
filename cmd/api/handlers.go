package main

import (
	"fmt"
	"net/http"
	"pipefy-integration/internal/repository"
	"pipefy-integration/internal/service"
	"time"
)

type createCardRequest struct {
	NomeCliente     string   `json:"cliente_nome" validate:"required"`
	EmailCliente    string   `json:"cliente_email" validate:"required,email"`
	TipoSolicitacao string   `json:"tipo_solicitacao" validate:"required"`
	ValorPatrimonio *float64 `json:"valor_patrimonio" validate:"required,gte=0"`
}

func (app *application) createCardHandler(w http.ResponseWriter, r *http.Request) {
	var payload createCardRequest
	if err := app.readJSON(w, r, &payload); err != nil {
		app.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to read request body: %s", err.Error()),
		})
		return
	}

	if err := app.validator.Validate(payload); err != nil {
		app.writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
		return
	}

	// persistir no banco local
	client := service.CreateUserData{
		NomeCliente:     payload.NomeCliente,
		EmailCliente:    payload.EmailCliente,
		TipoSolicitacao: payload.TipoSolicitacao,
		ValorPatrimonio: *payload.ValorPatrimonio,
	}
	id, err := app.databaseService.CreateUser(r.Context(), client)
	if err != nil {
		switch err {
		case repository.ErrClientAlreadyExists:
			app.writeJSON(w, http.StatusConflict, map[string]string{
				"error": "client already exists",
			})
			return
		default:
			app.writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
			return
		}
	}

	// mapear create no pipefy

	app.writeJSON(w, http.StatusAccepted, map[string]string{
		"client_id": id,
	})
}

type updateCardRequest struct {
	EventID      string    `json:"event_id" validate:"required"`
	CardID       string    `json:"card_id" validate:"required"`
	EmailCliente string    `json:"cliente_email" validate:"required,email"`
	Timestamp    time.Time `json:"timestamp" validate:"required"`
}

func (app *application) updateCardHandler(w http.ResponseWriter, r *http.Request) {
	var payload updateCardRequest
	if err := app.readJSON(w, r, &payload); err != nil {
		app.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to read request body: %s", err.Error()),
		})
		return
	}

	if err := app.validator.Validate(payload); err != nil {
		app.writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
		return
	}

	event := service.ProcessEventData{
		EventID:      payload.EventID,
		CardID:       payload.CardID,
		EmailCliente: payload.EmailCliente,
		Timestamp:    payload.Timestamp,
	}

	// mapear update no pipefy

	// alterar no banco local
	err := app.databaseService.ProcessEvent(r.Context(), event)
	if err != nil {
		switch err {
		case repository.ErrEventAlreadyProcessed:
			app.writeJSON(w, http.StatusConflict, map[string]string{
				"error": "event already processed",
			})
			return
		case repository.ErrClientNotFound:
			app.writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "client for event not found",
			})
			return
		default:
			app.writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}
	}

	app.writeJSON(w, http.StatusOK, map[string]string{
		"processed": event.EventID,
	})
}

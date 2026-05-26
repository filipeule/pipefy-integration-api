package repository

import (
	"context"
	"errors"
	"pipefy-integration/internal/models"
)

var (
	ErrEventAlreadyProcessed = errors.New("event already processed")
	ErrClientAlreadyExists   = errors.New("cliente already exists")
	ErrClientNotFound        = errors.New("client not found")
)

type DatabaseStore interface {
	CreateClient(ctx context.Context, cliente *models.Cliente) error
	ProcessEvent(ctx context.Context, event models.Event, fn ProcessEventFunc) (*models.PrioridadeCliente, error)
	Close() error
}

type ProcessEventFunc func(ctx context.Context, cliente models.Cliente) (models.PrioridadeCliente, error)

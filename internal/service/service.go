package service

import (
	"context"
	"math"
	"pipefy-integration/internal/models"
	"pipefy-integration/internal/repository"
	"time"

	"github.com/google/uuid"
)

type DatabaseService struct {
	store repository.DatabaseStore
}

func NewDatabaseService(store repository.DatabaseStore) *DatabaseService {
	return &DatabaseService{
		store: store,
	}
}

type CreateUserData struct {
	NomeCliente     string
	EmailCliente    string
	TipoSolicitacao string
	ValorPatrimonio float64
}

func (s *DatabaseService) CreateUser(ctx context.Context, user CreateUserData) (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}

	patrimonioInCents := int(math.Round(user.ValorPatrimonio * 1000))

	cliente := models.Cliente{
		ID:              id,
		NomeCliente:     user.NomeCliente,
		EmailCliente:    user.EmailCliente,
		TipoSolicitacao: user.TipoSolicitacao,
		ValorPatrimonio: models.Money(patrimonioInCents),
		Prioridade:      models.PrioridadeNormal,
		Status:          models.StatusAguardandoAnalise,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	err = s.store.CreateClient(ctx, &cliente)
	if err != nil {
		return "", err
	}

	return id.String(), nil
}

type ProcessEventData struct {
	EventID      string
	CardID       string
	EmailCliente string
	Timestamp    time.Time
}

func (s *DatabaseService) ProcessEvent(ctx context.Context, eventData ProcessEventData) error {
	event := models.Event{
		EventID:      eventData.EventID,
		CardID:       eventData.CardID,
		EmailCliente: eventData.EmailCliente,
		Timestamp:    eventData.Timestamp,
	}

	return s.store.ProcessEvent(ctx, event, func(ctx context.Context, cliente models.Cliente) (models.PrioridadeCliente, error) {
		if cliente.ValorPatrimonio >= models.ClientNetWorthThreshold {
			return models.PrioridadeAlta, nil
		}

		return models.PrioridadeNormal, nil
	})
}

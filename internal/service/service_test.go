package service

import (
	"context"
	"errors"
	"pipefy-integration/internal/models"
	"pipefy-integration/internal/repository"
	"testing"
	"time"
)

type mockStore struct {
	createClientFn func(ctx context.Context, c *models.Cliente) error
	processEventFn func(ctx context.Context, event models.Event, fn repository.ProcessEventFunc) (*models.PrioridadeCliente, error)
}

func (m *mockStore) CreateClient(ctx context.Context, c *models.Cliente) error {
	return m.createClientFn(ctx, c)
}

func (m *mockStore) ProcessEvent(ctx context.Context, event models.Event, fn repository.ProcessEventFunc) (*models.PrioridadeCliente, error) {
	return m.processEventFn(ctx, event, fn)
}

func (m *mockStore) Close() error {
	return nil
}

func TestCreateUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       CreateUserData
		storeFn     func(ctx context.Context, c *models.Cliente) error
		wantErr     bool
		expectedErr error
	}{
		{
			name: "create client successfully",
			input: CreateUserData{
				NomeCliente:     "João Silva",
				EmailCliente:    "joao@example.com",
				TipoSolicitacao: "Atualização cadastral",
				ValorPatrimonio: 250000,
			},
			storeFn: func(ctx context.Context, c *models.Cliente) error {
				return nil
			},
			wantErr: false,
		},
		{
			name: "returns error if client already exists",
			input: CreateUserData{
				NomeCliente:     "João Silva",
				EmailCliente:    "joao@example.com",
				TipoSolicitacao: "Atualização cadastral",
				ValorPatrimonio: 250000,
			},
			storeFn: func(ctx context.Context, c *models.Cliente) error {
				return repository.ErrClientAlreadyExists
			},
			wantErr:     true,
			expectedErr: repository.ErrClientAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewDatabaseService(&mockStore{createClientFn: tt.storeFn})
			id, err := svc.CreateUser(context.Background(), tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
					t.Errorf("wanted %v, got %v", tt.expectedErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, but got: %v", err)
			}

			if id == "" {
				t.Fatal("id should not be empty")
			}
		})
	}
}

func TestProcessEvent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		input             ProcessEventData
		clientePatrimonio models.Money
		storeErr          error
		wantErr           bool
		expectedErr       error
		expectedPriority  models.PrioridadeCliente
	}{
		{
			name: "networth above the threshold -> high priority",
			input: ProcessEventData{
				EventID:      "evt_001",
				CardID:       "card_001",
				EmailCliente: "joao@example.com",
				Timestamp:    time.Now(),
			},
			clientePatrimonio: models.Money(200000000), // R$ 200.000
			expectedPriority:  models.PrioridadeAlta,
		},
		{
			name: "networth below the threshold -> normal priority",
			input: ProcessEventData{
				EventID:      "evt_001",
				CardID:       "card_001",
				EmailCliente: "joao@example.com",
				Timestamp:    time.Now(),
			},
			clientePatrimonio: models.Money(199999000), // R$ 199.999
			expectedPriority:  models.PrioridadeNormal,
		},
		{
			name: "networth equal the threshold -> high priority",
			input: ProcessEventData{
				EventID:      "evt_001",
				CardID:       "card_001",
				EmailCliente: "joao@example.com",
				Timestamp:    time.Now(),
			},
			clientePatrimonio: models.ClientNetWorthThreshold,
			expectedPriority:  models.PrioridadeAlta,
		},
		{
			name: "returns error if event is duplicated",
			input: ProcessEventData{
				EventID:      "evt_001",
				CardID:       "card_001",
				EmailCliente: "joao@example.com",
				Timestamp:    time.Now(),
			},
			storeErr:    repository.ErrEventAlreadyProcessed,
			wantErr:     true,
			expectedErr: repository.ErrEventAlreadyProcessed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockStore{
				processEventFn: func(ctx context.Context, event models.Event, fn repository.ProcessEventFunc) (*models.PrioridadeCliente, error) {
					if tt.storeErr != nil {
						return nil, tt.storeErr
					}

					cliente := models.Cliente{ValorPatrimonio: tt.clientePatrimonio}
					priority, err := fn(ctx, cliente)

					return &priority, err
				},
			}

			svc := NewDatabaseService(store)
			priority, err := svc.ProcessEvent(context.Background(), tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, but got nil")
				}

				if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
					t.Errorf("wanted %v, got %v", tt.expectedErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if *priority != tt.expectedPriority {
				t.Errorf("priority expected '%s', but got '%s'", tt.expectedPriority, *priority)
			}
		})
	}
}

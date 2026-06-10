package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"pipefy-integration/internal/models"
	pgstore "pipefy-integration/internal/repository/postgres"
	"pipefy-integration/internal/service"
	"pipefy-integration/internal/validate"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	testApp  *application
	testPool *pgxpool.Pool
)

func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Short() {
		os.Exit(m.Run())
	}

	os.Exit(prepareTests(m))
}

func prepareTests(m *testing.M) int {
	if testing.Short() {
		return m.Run()
	}

	ctx := context.Background()

	container, err := postgres.Run(ctx,
		"postgres:18.4-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		log.Fatalf("failed to start container: %v", err)
	}
	defer container.Terminate(ctx)

	connStr, _ := container.ConnectionString(ctx, "sslmode=disable")
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("error connecting to container: %v", err)
	}
	defer pool.Close()

	// migrations
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS clientes (
			id UUID PRIMARY KEY,
			nome VARCHAR(255) NOT NULL,
			email VARCHAR(255) UNIQUE NOT NULL,
			tipo_solicitacao VARCHAR(255) NOT NULL,
			valor_patrimonio BIGINT NOT NULL,
			prioridade VARCHAR(50) NOT NULL,
			status VARCHAR(50) NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		);

		CREATE TABLE processed_events (
			event_id VARCHAR(50) PRIMARY KEY,
			card_id VARCHAR(50) NOT NULL,
			client_email VARCHAR(255) NOT NULL,
			event_time TIMESTAMPTZ NOT NULL,
			processed_at TIMESTAMPTZ DEFAULT NOW()
		);
	`)
	if err != nil {
		log.Printf("failed to create tables: %v", err)
		return 1
	}

	store, err := pgstore.NewStore(ctx, connStr)
	if err != nil {
		log.Printf("failed to initialize postgres store: %v", err)
		return 1
	}

	dbSvc := service.NewDatabaseService(store)
	pfSvc, _ := service.NewPipefyService("http://pipefy/graphql", "dummy-token")
	v := validate.New()

	app := &application{
		databaseService: dbSvc,
		pipefyService:   pfSvc,
		validator:       v,
	}

	testApp = app
	testPool = pool

	return m.Run()
}

func TestCreateClienteIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cleanDB(t)

	mux := testApp.mount()

	tests := []struct {
		name           string
		body           map[string]any
		expectedStatus int
	}{
		{
			name: "create client successfully - returns 202",
			body: map[string]any{
				"cliente_nome":     "João Silva",
				"cliente_email":    "joao@example.com",
				"tipo_solicitacao": "Atualização cadastral",
				"valor_patrimonio": 250000,
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name: "invalid field (email) - returns 400",
			body: map[string]any{
				"cliente_nome":     "João Silva",
				"cliente_email":    "nao-é-email",
				"tipo_solicitacao": "Atualização cadastral",
				"valor_patrimonio": 250000,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "duplicated user - returns 409",
			body: map[string]any{
				"cliente_nome":     "João Silva",
				"cliente_email":    "joao@example.com",
				"tipo_solicitacao": "Atualização cadastral",
				"valor_patrimonio": 250000,
			},
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)

			req := httptest.NewRequest(http.MethodPost, "/clientes", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("wanted %d, got %d — body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestProcessEventIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cleanDB(t)
	cliente := createTestCliente(t)

	mux := testApp.mount()

	tests := []struct {
		name           string
		body           map[string]any
		expectedStatus int
	}{
		{
			name: "process event and defines priority - returns 200",
			body: map[string]any{
				"event_id":      "evt_001",
				"card_id":       "card_001",
				"cliente_email": cliente.EmailCliente,
				"timestamp":     "2026-05-18T12:00:00Z",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "duplicated event - returns 409",
			body: map[string]any{
				"event_id":      "evt_001",
				"card_id":       "card_001",
				"cliente_email": cliente.EmailCliente,
				"timestamp":     "2026-05-18T12:00:00Z",
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name: "client not found - returns 404",
			body: map[string]any{
				"event_id":      "evt_002",
				"card_id":       "card_002",
				"cliente_email": "noclient@example.com",
				"timestamp":     "2026-05-18T12:00:00Z",
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)

			req := httptest.NewRequest(http.MethodPost, "/webhooks/pipefy/card-updated", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("wanted %d, got %d — body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestClientCreatedAndProcessedE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test")
	}

	cleanDB(t)
	mux := testApp.mount()

	// create client
	createBody, _ := json.Marshal(map[string]any{
		"cliente_nome":     "João Silva",
		"cliente_email":    "joao@example.com",
		"tipo_solicitacao": "Atualização cadastral",
		"valor_patrimonio": 250000,
	})

	req := httptest.NewRequest(http.MethodPost, "/clientes", bytes.NewBuffer(createBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("wanted 202 accepted, got %d — body: %s", w.Code, w.Body.String())
	}

	// process event
	eventBody, _ := json.Marshal(map[string]any{
		"event_id":      "evt_e2e_001",
		"card_id":       "card_e2e_001",
		"cliente_email": "joao@example.com",
		"timestamp":     "2026-05-26T12:00:00Z",
	})

	req = httptest.NewRequest(http.MethodPost, "/webhooks/pipefy/card-updated", bytes.NewBuffer(eventBody))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("wanted 200 ok, got %d — body: %s", w.Code, w.Body.String())
	}

	// check if priorty is correctly defined
	var priority string
	err := testPool.QueryRow(context.Background(),
		"SELECT prioridade FROM clientes WHERE email = $1",
		"joao@example.com",
	).Scan(&priority)
	if err != nil {
		t.Fatalf("failed to get priority: %v", err)
	}

	if priority != string(models.PrioridadeAlta) {
		t.Errorf("wanted %s, got %s", models.PrioridadeAlta, priority)
	}
}

func cleanDB(t *testing.T) {
	t.Helper()

	_, err := testPool.Exec(context.Background(),
		"TRUNCATE TABLE clientes, processed_events RESTART IDENTITY CASCADE",
	)
	if err != nil {
		t.Fatalf("fail to clean db: %v", err)
	}
}

func createTestCliente(t *testing.T) models.Cliente {
	t.Helper()

	id, _ := uuid.NewV7()
	cliente := models.Cliente{
		ID:              id,
		NomeCliente:     "João Silva",
		EmailCliente:    "joao@example.com",
		TipoSolicitacao: "Atualização cadastral",
		ValorPatrimonio: models.Money(2500000), // R$ 25.000,00
		Prioridade:      models.PrioridadeNormal,
		Status:          models.StatusAguardandoAnalise,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	_, err := testPool.Exec(context.Background(), `
		INSERT INTO clientes (id, nome, email, tipo_solicitacao, valor_patrimonio, prioridade, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, cliente.ID, cliente.NomeCliente, cliente.EmailCliente, cliente.TipoSolicitacao,
		cliente.ValorPatrimonio, cliente.Prioridade, cliente.Status, cliente.CreatedAt, cliente.UpdatedAt)
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}

	return cliente
}

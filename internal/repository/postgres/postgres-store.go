package postgres

import (
	"context"
	"errors"
	"pipefy-integration/internal/models"
	"pipefy-integration/internal/repository"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	db *pgxpool.Pool
}

func NewStore(ctx context.Context, dsn string) (*PostgresStore, error) {
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping(ctx)
	if err != nil {
		return nil, err
	}

	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) CreateClient(ctx context.Context, cliente *models.Cliente) error {
	query := `INSERT INTO clientes VALUES($1, $2, $3, $4, $5, $6, $7)`
	_, err := s.db.Exec(
		ctx,
		query,
		cliente.ID,
		cliente.NomeCliente,
		cliente.EmailCliente,
		cliente.TipoSolicitacao,
		cliente.ValorPatrimonio,
		cliente.Prioridade,
		cliente.Status,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return repository.ErrClientAlreadyExists
		}
		return err
	}

	return nil
}

func (s *PostgresStore) ProcessEvent(
	ctx context.Context, event models.Event, eventFunc repository.ProcessEventFunc,
) (*models.PrioridadeCliente, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// verifica idempotência no banco para não atuar em eventos já processados
	insertQuery := `INSERT INTO processed_events VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING`
	ct, err := tx.Exec(ctx, insertQuery, event.EventID, event.CardID, event.EmailCliente, event.Timestamp)
	if err != nil {
		return nil, err
	}

	if ct.RowsAffected() == 0 {
		return nil, repository.ErrEventAlreadyProcessed
	}

	// busca o cliente e o valor do seu patrimonio
	var valorPatrimonio int64
	clientQuery := `
		SELECT
			valor_patrimonio
		FROM
			clientes
		WHERE
			email = $1`
	err = tx.QueryRow(ctx, clientQuery, event.EmailCliente).Scan(&valorPatrimonio)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrClientNotFound
		}
		return nil, err
	}

	cliente := models.Cliente{
		ValorPatrimonio: models.Money(valorPatrimonio),
	}

	// armazena a prioridade decidida pelo service
	newPriority, err := eventFunc(ctx, cliente)
	if err != nil {
		return nil, err
	}

	// atualiza o cliente com a prioridade definida e status processado
	updateQuery := `UPDATE clientes SET prioridade = $1, status = $2 WHERE email = $3`
	_, err = tx.Exec(ctx, updateQuery, newPriority, models.StatusProcessado, event.EmailCliente)
	if err != nil {
		return nil, err
	}

	return &newPriority, tx.Commit(ctx)
}

func (s *PostgresStore) Close() error {
	s.db.Close()
	return nil
}

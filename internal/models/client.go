package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	PrioridadeNormal PrioridadeCliente = "prioridade_normal"
	PrioridadeAlta   PrioridadeCliente = "prioridade_alta"
)

const (
	StatusAguardandoAnalise Status = "Aguardando Análise"
	StatusProcessado        Status = "Processado"
)

const (
	ClientNetWorthThreshold Money = 200000000
)

type PrioridadeCliente string

type Status string

type Cliente struct {
	ID              uuid.UUID
	NomeCliente     string
	EmailCliente    string
	TipoSolicitacao string
	ValorPatrimonio Money
	Prioridade      PrioridadeCliente
	Status          Status
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

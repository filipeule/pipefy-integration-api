package service

import "time"

type CreateUserData struct {
	NomeCliente     string
	EmailCliente    string
	TipoSolicitacao string
	ValorPatrimonio float64
}

type ProcessEventData struct {
	EventID      string
	CardID       string
	EmailCliente string
	Timestamp    time.Time
}
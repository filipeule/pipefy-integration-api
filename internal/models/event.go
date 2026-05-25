package models

import "time"

type Event struct {
	EventID      string
	CardID       string
	EmailCliente string
	Timestamp    time.Time
}

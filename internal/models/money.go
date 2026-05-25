package models

import "fmt"

type Money int64

func (m Money) String() string {
	value := m / 100
	cents := m % 100
	return fmt.Sprintf("$ %d,%02d", value, cents)
}

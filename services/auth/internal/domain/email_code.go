package domain

import "time"

type EmailCode struct {
	UserID    int64
	Purpose   string
	CodeHash  []byte
	Attempts  int
	ExpiresAt time.Time
	CreatedAt time.Time
}

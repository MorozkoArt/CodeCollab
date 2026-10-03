package domain

import "time"

type User struct {
	ID              int64
	Username        string
	Email           string
	Password        string
	EmailVerifiedAt *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

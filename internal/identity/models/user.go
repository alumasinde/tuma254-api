package models

import "time"

type User struct {
	ID           string
	Email        *string
	Phone        *string
	FirstName    string
	LastName     string
	PasswordHash string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type PublicUser struct {
	ID        string   `json:"id"`
	Email     *string  `json:"email,omitempty"`
	Phone     *string  `json:"phone,omitempty"`
	FirstName string   `json:"first_name"`
	LastName  string   `json:"last_name"`
	Status    string   `json:"status"`
	Roles     []string `json:"roles"`
}

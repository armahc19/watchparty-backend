package models

type User struct {
	ID       string `json:"id" db:"id"`
	Email    string `json:"email" db:"email"`
	Username string `json:"username" db:"username"`
}

type AuthUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}
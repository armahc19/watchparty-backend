package models

import (
	"time"
)

type Party struct {
	ID          string    `json:"id" db:"id"`
	HostID      string    `json:"host_id" db:"host_id"`
	Title       string    `json:"title" db:"title"`
	Description string    `json:"description" db:"description"`
	RoomCode    string    `json:"room_code" db:"room_code"`
	IsLive      bool      `json:"is_live" db:"is_live"`
	MaxViewers  int       `json:"max_viewers" db:"max_viewers"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
	ActivityType string   `json:"activity_type" db:"activity_type"` // Add this
}

type CreatePartyRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	MaxViewers  int    `json:"max_viewers" binding:"required,min=1,max=50"`
	ActivityType string `json:"activity_type" binding:"required,oneof=movie music browse game"` // Add this
}

type JoinPartyRequest struct {
	RoomCode string `json:"room_code" binding:"required"`
}

type PartyResponse struct {
	Party      *Party   `json:"party"`
	Host       *User    `json:"host"`
	ViewerCount int     `json:"viewer_count"`
	IsHost     bool     `json:"is_host"`
}
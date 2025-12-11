package handlers

import (
	//"fmt"
	"math/rand"
	"net/http"
	"time"

	"backend/database"
	"backend/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

)

func CreateParty(c *gin.Context) {
	userID := c.MustGet("userID").(string)
	
	var req models.CreatePartyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate unique room code
	roomCode := generateRoomCode()

	party := models.Party{
		ID:          uuid.New().String(),
		HostID:      userID,
		Title:       req.Title,
		Description: req.Description,
		RoomCode:    roomCode,
		IsLive:      false,
		MaxViewers:  req.MaxViewers,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		ActivityType: req.ActivityType, // Add this line
	}

	// Insert into database
	query := `
		INSERT INTO parties (id, host_id, title, description, room_code, is_live, max_viewers, created_at, updated_at,activity_type)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := database.DB.Exec(c, query,
		party.ID, party.HostID, party.Title, party.Description,
		party.RoomCode, party.IsLive, party.MaxViewers,
		party.CreatedAt, party.UpdatedAt, party.ActivityType,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create party"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Party created successfully",
		"party": gin.H{
			"id":          party.ID,
			"title":       party.Title,
			"room_code":   party.RoomCode,
			"max_viewers": party.MaxViewers,
			"created_at":  party.CreatedAt,
			"activity_type": party.ActivityType,
		},
	})
}

func JoinParty(c *gin.Context) {
	userID := c.MustGet("userID").(string)
	
	var req models.JoinPartyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get party by room code
	var party models.Party
	query := `
		SELECT id, host_id, title, description, room_code, is_live, max_viewers, created_at, updated_at, activity_type
		FROM parties WHERE room_code = $1
	`
	err := database.DB.QueryRow(c, query, req.RoomCode).Scan(
		&party.ID, &party.HostID, &party.Title, &party.Description,
		&party.RoomCode, &party.IsLive, &party.MaxViewers,
		&party.CreatedAt, &party.UpdatedAt, &party.ActivityType,
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Party not found"})
		return
	}

	// Check if party is full
	var currentViewers int
	countQuery := `SELECT COUNT(*) FROM party_participants WHERE party_id = $1`
	database.DB.QueryRow(c, countQuery, party.ID).Scan(&currentViewers)

	if currentViewers >= party.MaxViewers {
		c.JSON(http.StatusForbidden, gin.H{"error": "Party is full"})
		return
	}

	// Add user to party participants
	participantQuery := `
		INSERT INTO party_participants (id, party_id, user_id, joined_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (party_id, user_id) DO NOTHING
	`
	_, err = database.DB.Exec(c, participantQuery,
		uuid.New().String(), party.ID, userID, time.Now(),
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to join party"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Successfully joined party",
		"party": gin.H{
			"id":        party.ID,
			"title":     party.Title,
			"room_code": party.RoomCode,
			"is_live":   party.IsLive,
			"host_id":   party.HostID,
			"activity_type": party.ActivityType,
		},
	})
}

func GetParty(c *gin.Context) {
	userID := c.MustGet("userID").(string)
	roomID := c.Param("roomId")

	var party models.Party
	query := `
		SELECT id, host_id, title, description, room_code, is_live, max_viewers, created_at, updated_at, activity_type
		FROM parties WHERE id = $1
	`
	err := database.DB.QueryRow(c, query, roomID).Scan(
		&party.ID, &party.HostID, &party.Title, &party.Description,
		&party.RoomCode, &party.IsLive, &party.MaxViewers,
		&party.CreatedAt, &party.UpdatedAt, &party.ActivityType,
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Party not found"})
		return
	}

	// Get host info
	var host models.User
	hostQuery := `SELECT id, email, username FROM profiles WHERE id = $1`
	database.DB.QueryRow(c, hostQuery, party.HostID).Scan(&host.ID, &host.Email, &host.Username)

	// Get viewer count
	var viewerCount int
	countQuery := `SELECT COUNT(*) FROM party_participants WHERE party_id = $1`
	database.DB.QueryRow(c, countQuery, party.ID).Scan(&viewerCount)

	// Check if current user is host
	isHost := party.HostID == userID

	response := models.PartyResponse{
		Party:      &party,
		Host:       &host,
		ViewerCount: viewerCount,
		IsHost:     isHost,
		
	}

	c.JSON(http.StatusOK, response)
}

func generateRoomCode() string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, 6)
	for i := range code {
		code[i] = chars[rand.Intn(len(chars))]
	}
	return string(code)
}
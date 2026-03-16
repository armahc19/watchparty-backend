package main

import (
	"log"
	"os"
	"fmt"
	"time"

	"backend/database"
	"backend/handlers"
	"backend/middleware"
	"backend/websocket"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)


var allowedOrigins = map[string]bool{
	"https://watchparty-app-drab.vercel.app": true,
	"https://watchparty.site":                true,
	"https://www.watchparty.site":            true,
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		origin := c.Request.Header.Get("Origin")

		if allowedOrigins[origin] {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set(
			"Access-Control-Allow-Headers",
			"Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With",
		)
		c.Writer.Header().Set(
			"Access-Control-Allow-Methods",
			"POST, OPTIONS, GET, PUT, DELETE",
		)

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize database
	if err := database.InitSupabase(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.DB.Close()

	// After database initialization
	// main.go — THIS KEEPS YOUR SERVER ALIVE FOREVER
wsHub := websocket.NewHub()

go func() {
    for {
        fmt.Println("WebSocket hub starting...")
        wsHub.Run()  // This blocks until panic
        fmt.Println("WebSocket hub crashed! Restarting in 1 second...")
        time.Sleep(1 * time.Second)
        // Create new hub if old one died
        wsHub = websocket.NewHub()
    }
}()

	// Create Gin router
	r := gin.Default()

	// Add CORS middleware - MUST be before all other middleware
	r.Use(CORSMiddleware())

	// Routes
	api := r.Group("/api")
	{
		// Party routes with auth
		party := api.Group("/party")
		party.Use(middleware.AuthMiddleware())
		{
			party.POST("/create", handlers.CreateParty)
			party.POST("/join", handlers.JoinParty)
			party.GET("/:roomId", handlers.GetParty)
		}

		// Health check (no auth required)
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok", "service": "watchparty-backend"})
		})

		// WebSocket endpoint
		// Remove AuthMiddleware from WebSocket route since we handle auth manually
		api.GET("/ws/:roomId", func(c *gin.Context) {
			websocket.ServeWs(wsHub, c)
		})

		
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("Server running on port %s", port)
	log.Fatal(r.Run(":" + port))
}


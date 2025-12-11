package websocket

import (
    "fmt"
    "log"
    "net/http"
    "os"

    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v4"
    "github.com/gorilla/websocket"
)

type Client struct {
    hub     *Hub
    conn    *websocket.Conn
    send    chan Message
    roomID  string
    userID  string
    closing bool  // Add flag to track if we're closing
}

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true // For development only
    },
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
}

type Claims struct {
    UserID string `json:"sub"`
    Email  string `json:"email"`
    jwt.StandardClaims
}

func (c *Client) readPump() {
    defer func() {
        c.conn.Close()
        // DO NOT send to unregister here — safeClose() does it
    }()

    for {
        var msg Message
        err := c.conn.ReadJSON(&msg)
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                log.Printf("WebSocket error: %v", err)
            }
            break
        }

        msg.UserID = c.userID
        msg.RoomID = c.roomID
        c.hub.broadcast <- msg
    }
}

func (c *Client) writePump() {
    defer func() {
        c.conn.Close()
    }()

    for {
        select {
        case message, ok := <-c.send:
            if !ok {
                // Channel closed, send close message to client
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }

            err := c.conn.WriteJSON(message)
            if err != nil {
                log.Printf("Write error: %v", err)
                return
            }
        }
    }
}

// Safe function to close client
func (c *Client) safeClose() {
    c.closing = true

    // Send close message to client
    c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

    // Close connection immediately
    c.conn.Close()

    // Safely close send channel
    select {
    case <-c.send:
    default:
        close(c.send)
    }

    // UNREGISTER FROM HUB — THIS WAS MISSING!
    c.hub.unregister <- c
}

func ServeWs(hub *Hub, ctx *gin.Context) {
    roomID := ctx.Param("roomId")
    token := ctx.Query("token")
    if token == "" {
        ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Token required"})
        return
    }

    userID, err := verifyToken(token)
    if err != nil {
        ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
        return
    }

    conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
    if err != nil {
        log.Println("WebSocket upgrade failed:", err)
        return
    }

    // KICK OLD CONNECTION — CLEAN & SAFE
    hub.mu.Lock()
    oldClientToKick := (*Client)(nil)
    if roomClients, exists := hub.clients[roomID]; exists {
        for oc := range roomClients {
            if oc.userID == userID {
                oldClientToKick = oc
                break
            }
        }
        if oldClientToKick != nil {
            delete(roomClients, oldClientToKick)
            if len(roomClients) == 0 {
                delete(hub.clients, roomID)
            }
        }
    }
    hub.mu.Unlock()

    // If we found an old client, close it AFTER releasing the lock
    if oldClientToKick != nil {
        fmt.Printf("Kicking old connection for user %s\n", userID)
        oldClientToKick.safeClose()
    }

    // CREATE NEW CLIENT
    client := &Client{
        hub:     hub,
        conn:    conn,
        send:    make(chan Message, 256),
        roomID:  roomID,
        userID:  userID,
        closing: false,
    }

    // Register
    hub.register <- client

    // Start pumps
    go client.writePump()
    go client.readPump()

    fmt.Printf("User %s connected to room %s\n", userID, roomID)
}

func verifyToken(tokenString string) (string, error) {
    secret := os.Getenv("SUPABASE_JWT_SECRET")
    if secret == "" {
        return "", fmt.Errorf("JWT secret not configured")
    }

    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return []byte(secret), nil
    })

    if err != nil || !token.Valid {
        return "", fmt.Errorf("invalid token")
    }

    if claims, ok := token.Claims.(*Claims); ok {
        return claims.UserID, nil
    }

    return "", fmt.Errorf("invalid token claims")
}
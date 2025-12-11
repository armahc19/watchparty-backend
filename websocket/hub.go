// websocket/hub.go
package websocket

import "sync"
import "fmt"

type Message struct {
    RoomID    string  `json:"room_id"`
    Type      string  `json:"type"`
    Timestamp *float64 `json:"timestamp,omitempty"`
    IsPlaying *bool   `json:"isPlaying,omitempty"`
    UserID    string  `json:"user_id,omitempty"`
    Message   *string `json:"message,omitempty"`
    Emoji     *string `json:"emoji,omitempty"`
    User      *string `json:"user,omitempty"`
    FileData *struct {
        FileID   string `json:"file_id"`
        FileName string `json:"file_name"`
        FileType string `json:"file_type"`
        Duration int    `json:"duration"`
    } `json:"fileData,omitempty"`
}

type Hub struct {
    clients    map[string]map[*Client]bool // roomID -> clients
    broadcast  chan Message
    register   chan *Client
    unregister chan *Client
    mu         sync.RWMutex
}

func NewHub() *Hub {
    return &Hub{
        clients:    make(map[string]map[*Client]bool),
        broadcast:  make(chan Message),
        register:   make(chan *Client),
        unregister: make(chan *Client),
    }
}

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.mu.Lock()
            if h.clients[client.roomID] == nil {
                h.clients[client.roomID] = make(map[*Client]bool)
            }
            h.clients[client.roomID][client] = true
            h.mu.Unlock()
            fmt.Printf("Registered client for user %s in room %s. Total in room: %d\n", 
                client.userID, client.roomID, len(h.clients[client.roomID]))

        case client := <-h.unregister:
            h.mu.Lock()
            if clients, ok := h.clients[client.roomID]; ok {
                // Only process if client still exists in map
                if _, exists := clients[client]; exists {
                    // Safe channel close
                    select {
                    case <-client.send:
                        // Channel already closed
                    default:
                        close(client.send)
                    }
                    
                    delete(clients, client)
                    
                    fmt.Printf("Unregistered client for user %s in room %s. Remaining: %d\n", 
                        client.userID, client.roomID, len(clients))
                    
                    if len(clients) == 0 {
                        delete(h.clients, client.roomID)
                        fmt.Printf("Room %s is now empty, removed from hub\n", client.roomID)
                    }
                }
            }
            h.mu.Unlock()

        case message := <-h.broadcast:
            h.mu.RLock()
            roomClients, exists := h.clients[message.RoomID]
            if exists {
                fmt.Printf("Broadcasting to %d clients in room %s\n", len(roomClients), message.RoomID)
                for client := range roomClients {
                    select {
                    case client.send <- message:
                        // Message sent successfully
                    default:
                        // Client's send buffer is full, close connection
                        go func(c *Client) {
                            h.unregister <- c
                            c.safeClose()
                        }(client)
                    }
                }
            }
            h.mu.RUnlock()
        }
    }
}
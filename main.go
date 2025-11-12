package main

import (
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // allow all origins
	},
}

// Track connected clients
var (
	clients   = make(map[*websocket.Conn]bool)
	mutex     sync.Mutex
	broadcast = make(chan map[string]interface{}) // open JSON object
)

// Handle WebSocket connections
func handleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	mutex.Lock()
	clients[conn] = true
	mutex.Unlock()

	// Listen for messages from this client
	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			// Client disconnected
			mutex.Lock()
			delete(clients, conn)
			mutex.Unlock()
			return
		}

		// Broadcast the message
		broadcast <- msg
	}
}

// Broadcast messages to all clients
func broadcaster() {
	for {
		msg := <-broadcast
		mutex.Lock()
		for conn := range clients {
			if err := conn.WriteJSON(msg); err != nil {
				conn.Close()
				delete(clients, conn)
			}
		}
		mutex.Unlock()
	}
}

func main() {
	http.HandleFunc("/ws", handleConnections)
	go broadcaster()

	// Use Render-assigned port
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("WebSocket server running on port:", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/websocket"
)

// Upgrade HTTP connection to WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // allow all origins
	},
}

type Client struct {
	conn *websocket.Conn
	name string
}

var (
	clients   = make(map[*websocket.Conn]Client)
	mutex     sync.Mutex
	broadcast = make(chan string)
)

// Handle incoming WebSocket connections
func handleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	// Ask for username
	if err := conn.WriteMessage(websocket.TextMessage, []byte("Enter your name:")); err != nil {
		return
	}

	_, msg, err := conn.ReadMessage()
	if err != nil {
		return
	}
	name := string(msg)

	mutex.Lock()
	clients[conn] = Client{conn: conn, name: name}
	mutex.Unlock()

	broadcast <- fmt.Sprintf("%s has joined the chat", name)

	// Listen for messages
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			mutex.Lock()
			broadcast <- fmt.Sprintf("%s has left the chat", clients[conn].name)
			delete(clients, conn)
			mutex.Unlock()
			return
		}
		broadcast <- fmt.Sprintf("%s: %s", name, msg)
	}
}

// Send messages to all clients
func broadcaster() {
	for {
		msg := <-broadcast
		mutex.Lock()
		for conn := range clients {
			if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
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

	// ✅ Use Render’s assigned port
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // fallback for local dev
	}

	fmt.Println("WebSocket chat server running on port:", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

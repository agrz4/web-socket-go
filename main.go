package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan Message)
var messages []Message // Menyimpan riwayat pesan
var mu sync.RWMutex

type Message struct {
	Username string `json:"username"`
	Message  string `json:"message"`
	Type     string `json:"type"` // "chat", "clear", "system", "history"
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	http.ServeFile(w, r, "index.html")
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Upgrade error: %v", err)
		return
	}
	defer ws.Close()

	mu.Lock()
	clients[ws] = true
	mu.Unlock()
	log.Println("Client connected")

	// Kirim history pesan ke client baru
	mu.RLock()
	for _, msg := range messages {
		if err := ws.WriteJSON(msg); err != nil {
			log.Printf("Send history error: %v", err)
			break
		}
	}
	mu.RUnlock()

	for {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("Read error: %v", err)
			mu.Lock()
			delete(clients, ws)
			mu.Unlock()
			break
		}

		// Cek perintah khusus
		if msg.Message == "/clear" {
			mu.Lock()
			messages = []Message{} // Kosongkan riwayat
			mu.Unlock()

			// Kirim ke semua klien
			broadcast <- Message{
				Username: msg.Username,
				Message:  "Obrolan dibersihkan.",
				Type:     "clear",
			}
			continue
		}

		// Simpan pesan normal
		msg.Type = "chat"
		mu.Lock()
		messages = append(messages, msg)
		mu.Unlock()

		broadcast <- msg
	}
}

func handleMessages() {
	for {
		msg := <-broadcast
		mu.RLock()
		clientsCopy := make([]*websocket.Conn, 0, len(clients))
		for client := range clients {
			clientsCopy = append(clientsCopy, client)
		}
		mu.RUnlock()

		for _, client := range clientsCopy {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Printf("Write error: %v", err)
				client.Close()
				mu.Lock()
				delete(clients, client)
				mu.Unlock()
			}
		}
	}
}

func main() {
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", handleConnections)

	go handleMessages()

	log.Println("Server berjalan di http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

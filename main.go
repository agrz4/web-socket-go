package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// upgrader akan meng-upgrade HTTP request menjadi koneksi WebSocket.
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return r.Header.Get("Origin") == "yourdomain.com"
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var (
	// clients menyimpan semua koneksi WebSocket yang aktif.
	clients = make(map[*websocket.Conn]bool)

	// broadcast adalah channel untuk mengirim pesan ke semua klien.
	broadcast = make(chan []byte)

	// mutex digunakan untuk mengamankan akses ke map `clients`
	// dari race condition.
	mutex = &sync.Mutex{}
)

// wsHandler menangani permintaan WebSocket dari klien.
func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading to WebSocket: %v", err)
		return
	}
	defer conn.Close()

	// Menambahkan klien baru ke map.
	mutex.Lock()
	clients[conn] = true
	mutex.Unlock()
	log.Printf("New client connected: %v", conn.RemoteAddr())

	// Membaca pesan dari klien secara terus-menerus.
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Client disconnected or error reading message: %v", err)

			// Menghapus klien yang terputus dari map.
			mutex.Lock()
			delete(clients, conn)
			mutex.Unlock()
			break
		}
		// Mengirim pesan ke channel broadcast untuk didistribusikan.
		broadcast <- message
	}
}

// handleMessages mendengarkan pesan dari channel `broadcast`
// dan mengirimkannya ke semua klien yang terhubung.
func handleMessages() {
	for {
		message := <-broadcast

		mutex.Lock()
		for client := range clients {
			err := client.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Printf("Error writing message to client: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
		mutex.Unlock()
	}
}

func main() {
	// Menangani endpoint WebSocket
	http.HandleFunc("/ws", wsHandler)

	// Menjalankan goroutine untuk menangani pesan siaran
	go handleMessages()

	log.Println("WebSocket server started on :8080")
	fmt.Println("Access the WebSocket client at http://localhost:8080")

	// Menjalankan server HTTP
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}

package main

import (
	"log"
	"net/http"
	"text/template"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan Message)

type Message struct {
	Username string `json:"username"`
	Message  string `json:"message"`
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("error: %v", err)
		return
	}
	defer ws.Close()

	clients[ws] = true
	log.Printf("New client connected. Current number of clients: %d", len(clients))

	for {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, ws)
			log.Printf("Client disconnected. Current number of clients: %d", len(clients))
			break
		}
		log.Printf("Get Message from '%s': %s", msg.Username, msg.Message)
		broadcast <- msg
	}
}

func handleMessages() {
	for {
		msg := <-broadcast
		log.Printf("Broadcast the message '%s' to all clients", msg.Message)
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Printf("error: %v", err)
				client.Close()
				delete(clients, client)
				log.Printf("Client disconnected. Current number of clients: %d", len(clients))
			}
		}
	}
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	tmpl, err := template.ParseFiles("index.html")
	if err != nil {
		http.Error(w, "File not found", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func main() {
	// Mengatur rute untuk server
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", handleConnections)

	// Mulai goroutine untuk menangani pesan
	go handleMessages()

	log.Println("Server Golang sedang berjalan di http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

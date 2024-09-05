package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// Definir un actualizador de WebSocket para manejar las conexiones
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Permitir todas las solicitudes CORS
		return true
	},
}

// Mapa para almacenar las conexiones de WebSocket
var clients = make(map[*websocket.Conn]bool)
var mu sync.Mutex // Mutex para evitar condiciones de carrera

func main() {
	http.HandleFunc("/ws", handleConnections)

	port := ":3000"
	fmt.Printf("Servidor de WebSocket corriendo en http://localhost%s\n", port)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatalf("Error al iniciar el servidor: %v", err)
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	// Actualizar la conexión HTTP a una conexión WebSocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error al actualizar la conexión: %v", err)
		return
	}
	defer ws.Close()

	// Añadir el nuevo cliente al mapa
	mu.Lock()
	clients[ws] = true
	mu.Unlock()

	log.Println("Nuevo cliente conectado")

	for {
		// Leer el mensaje de audio desde el cliente
		_, message, err := ws.ReadMessage()
		if err != nil {
			log.Printf("Error al leer mensaje: %v", err)
			break
		}

		// Aquí puedes manejar el audio recibido
		log.Printf("Audio recibido: %d bytes", len(message))

		// Retransmitir el mensaje a todos los demás clientes conectados
		mu.Lock()
		for client := range clients {
			if client != ws {
				err := client.WriteMessage(websocket.BinaryMessage, message)
				if err != nil {
					log.Printf("Error al enviar mensaje a un cliente: %v", err)
					client.Close()
					delete(clients, client)
				}
			}
		}
		mu.Unlock()
	}

	// Eliminar al cliente cuando se desconecte
	mu.Lock()
	delete(clients, ws)
	mu.Unlock()

	log.Println("Cliente desconectado")
}

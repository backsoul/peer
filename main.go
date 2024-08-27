package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Permitir todas las solicitudes CORS
		return true
	},
}

// Lista de conexiones activas y un mutex para manejarlas de forma segura
var clients = make(map[*websocket.Conn]bool)
var clientsMutex = sync.Mutex{}

func main() {
	http.HandleFunc("/", handleConnections)

	port := ":3000"
	fmt.Printf("Servidor de audio en tiempo real corriendo en http://localhost%s\n", port)
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

	// Agregar el nuevo cliente a la lista de conexiones activas
	clientsMutex.Lock()
	clients[ws] = true
	clientsMutex.Unlock()

	log.Println("Nuevo cliente conectado")

	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			log.Printf("Error al leer mensaje: %v", err)
			break
		}

		// Emitir el mensaje de audio a todos los demás clientes
		err = broadcastMessage(ws, message)
		if err != nil {
			log.Printf("Error al emitir mensaje: %v", err)
			break
		}
	}

	// Remover el cliente de la lista de conexiones activas al desconectarse
	clientsMutex.Lock()
	delete(clients, ws)
	clientsMutex.Unlock()

	log.Println("Cliente desconectado")
}

func broadcastMessage(sender *websocket.Conn, message []byte) error {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	for client := range clients {
		if client != sender {
			err := client.WriteMessage(websocket.BinaryMessage, message)
			if err != nil {
				log.Printf("Error al enviar mensaje: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}

	return nil
}

package client

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	upgrader     = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	audioClients = make(map[*websocket.Conn]chan []byte)
	mu           sync.Mutex
)

// Maneja las conexiones WebSocket para enviar audio en tiempo real
func HandleAudioConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error al actualizar la conexión: %v", err)
		return
	}
	defer func() {
		mu.Lock()
		delete(audioClients, ws)
		mu.Unlock()
		ws.Close()
		log.Println("Cliente desconectado del envío de audio")
	}()

	audioChan := make(chan []byte, 10)

	// Añadir el cliente a la lista de conexiones activas
	mu.Lock()
	audioClients[ws] = audioChan
	mu.Unlock()

	log.Println("Nuevo cliente conectado para envío de audio")

	// Gorutina para enviar audio al cliente
	go func() {
		for audio := range audioChan {
			if err := ws.WriteMessage(websocket.BinaryMessage, audio); err != nil {
				log.Printf("Error al enviar audio: %v", err)
				break
			}
		}
	}()
}

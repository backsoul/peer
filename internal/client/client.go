package client

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	upgrader     = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	audioClients = make(map[*websocket.Conn]bool)
	mu           sync.Mutex
	broadcastCh  = make(chan broadcastMessage, 100) // Canal para manejar la retransmisión de audio
)

type broadcastMessage struct {
	messageType int
	message     []byte
	sender      *websocket.Conn
}

// Inicializar la retransmisión en un goroutine separado
func init() {
	go handleBroadcasts()
}

// Maneja las conexiones WebSocket para enviar y retransmitir audio en tiempo real
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

	mu.Lock()
	audioClients[ws] = true
	mu.Unlock()
	log.Println("Nuevo cliente conectado para envío de audio")

	// Escuchar mensajes del cliente
	for {
		messageType, message, err := ws.ReadMessage()
		if err != nil {
			log.Printf("Error al leer del WebSocket: %v", err)
			break
		}

		// Enviar el mensaje recibido al canal de retransmisión
		broadcastCh <- broadcastMessage{
			messageType: messageType,
			message:     message,
			sender:      ws,
		}
	}
}

// Maneja la retransmisión de audio a todos los clientes conectados
func handleBroadcasts() {
	for msg := range broadcastCh {
		broadcastAudio(msg.messageType, msg.message, msg.sender)
	}
}

// Retransmite el audio a todos los clientes conectados, excepto al remitente
func broadcastAudio(messageType int, message []byte, sender *websocket.Conn) {
	mu.Lock()
	defer mu.Unlock()

	for client := range audioClients {
		if client != sender {
			err := client.WriteMessage(messageType, message)
			if err != nil {
				log.Printf("Error al enviar mensaje a cliente: %v", err)
				client.Close()
				delete(audioClients, client)
			}
		}
	}
}

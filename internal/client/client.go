package client

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	upgrader     = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	audioClients = make(map[*websocket.Conn]bool)
	mu           sync.Mutex
)

type AudioMessage struct {
	Timestamp string `json:"timestamp"`
	AudioData []byte `json:"audioData"`
}

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

	for {
		messageType, message, err := ws.ReadMessage()
		if err != nil {
			log.Printf("Error al leer del WebSocket: %v", err)
			break
		}

		// Crear objeto con timestamp y audio
		audioMsg := AudioMessage{
			Timestamp: time.Now().Format(time.RFC3339),
			AudioData: message,
		}

		// Serializar el objeto a JSON
		audioMsgJSON, err := json.Marshal(audioMsg)
		if err != nil {
			log.Printf("Error al serializar el mensaje de audio: %v", err)
			continue
		}

		// Retransmitir el mensaje a todos los demás clientes
		broadcastAudio(messageType, audioMsgJSON, ws)
	}
}

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

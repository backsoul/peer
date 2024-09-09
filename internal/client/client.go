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

// AudioMessage define la estructura del mensaje que será enviado a los clientes
type AudioMessage struct {
	Audio     []byte `json:"audio"`
	Timestamp string `json:"timestamp"`
}

// Maneja las conexiones WebSocket para enviar y retransmitir audio en tiempo real
func HandleAudioConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error al actualizar la conexión: %v", err)
		return
	}

	// Asegurarse de cerrar la conexión cuando el cliente se desconecta
	defer func() {
		mu.Lock()
		delete(audioClients, ws)
		mu.Unlock()
		ws.Close()
		log.Println("Cliente desconectado del envío de audio")
	}()

	// Añadir el cliente a la lista de conexiones activas
	mu.Lock()
	audioClients[ws] = true
	mu.Unlock()

	log.Println("Nuevo cliente conectado para envío de audio")

	// Escuchar mensajes del cliente (esperamos recibir audio en formato binario)
	for {
		messageType, message, err := ws.ReadMessage()
		if err != nil {
			log.Printf("Error al leer del WebSocket: %v", err)
			break
		}

		// Crear un objeto AudioMessage con el audio y el timestamp actual
		audioMessage := AudioMessage{
			Audio:     message,
			Timestamp: time.Now().Format(time.RFC3339),
		}

		// Convertir el objeto AudioMessage a JSON
		messageJSON, err := json.Marshal(audioMessage)
		if err != nil {
			log.Printf("Error al convertir el mensaje a JSON: %v", err)
			continue
		}

		// Retransmitir el mensaje a todos los demás clientes
		broadcastAudio(messageType, messageJSON, ws)
	}
}

// Envía el audio recibido a todos los demás clientes conectados
func broadcastAudio(messageType int, messageJSON []byte, sender *websocket.Conn) {
	mu.Lock()
	defer mu.Unlock()

	for client := range audioClients {
		// No reenviar al cliente que envió el audio
		if client != sender {
			err := client.WriteMessage(messageType, messageJSON)
			if err != nil {
				log.Printf("Error al enviar mensaje a cliente: %v", err)
				client.Close()
				delete(audioClients, client)
			}
		}
	}
}

// Limpiar los clientes desconectados y mensajes pendientes cada intervalo de tiempo
func CleanUpOldMessages(interval time.Duration) {
	for {
		time.Sleep(interval)

		mu.Lock()
		for client := range audioClients {
			if err := client.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Cliente desconectado: %v", err)
				client.Close()
				delete(audioClients, client)
			}
		}
		mu.Unlock()
		log.Println("Limpieza de mensajes pendientes realizada")
	}
}

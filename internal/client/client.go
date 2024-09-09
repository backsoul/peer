package client

import (
	"encoding/binary"
	"fmt"
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

	// Añadir el cliente a la lista de conexiones activas
	mu.Lock()
	audioClients[ws] = true
	mu.Unlock()

	log.Println("Nuevo cliente conectado para envío de audio")

	for {
		// Escuchar mensajes del cliente (esperamos recibir audio en formato binario con timestamp)
		messageType, message, err := ws.ReadMessage()
		if err != nil {
			log.Printf("Error al leer del WebSocket: %v", err)
			break
		}

		// Obtener el timestamp del cliente (suponiendo que está al principio del mensaje)
		if len(message) < 8 {
			log.Printf("Mensaje recibido es demasiado corto para contener un timestamp")
			continue
		}

		// Leer los primeros 8 bytes como el timestamp (int64 en milisegundos)
		clientTimestamp := int64(binary.LittleEndian.Uint64(message[:8]))

		// Calcular el delay
		currentTime := time.Now().UnixMilli()
		delay := currentTime - clientTimestamp
		log.Printf("Delay calculado: %d ms", delay)

		// Retransmitir el audio (sin el timestamp) a los demás clientes
		broadcastAudio(messageType, message[8:], ws)

		// Enviar el delay de vuelta al cliente que envió el audio
		delayMessage := []byte(fmt.Sprintf("Delay: %d ms", delay))
		err = ws.WriteMessage(websocket.TextMessage, delayMessage)
		if err != nil {
			log.Printf("Error al enviar delay al cliente: %v", err)
			break
		}
	}
}

// Envía el audio recibido a todos los demás clientes conectados
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

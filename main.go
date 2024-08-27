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

// Estructura para manejar los canales
type Channel struct {
	clients map[*websocket.Conn]bool
	mutex   sync.Mutex
}

// Crear un canal de audio
var audioChannel = Channel{
	clients: make(map[*websocket.Conn]bool),
}

func main() {
	http.HandleFunc("/audio", handleAudioConnections)

	port := ":3000"
	fmt.Printf("Servidor de audio en tiempo real corriendo en http://localhost%s\n", port)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatalf("Error al iniciar el servidor: %v", err)
	}
}

func handleAudioConnections(w http.ResponseWriter, r *http.Request) {
	// Actualizar la conexión HTTP a una conexión WebSocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error al actualizar la conexión: %v", err)
		return
	}
	defer ws.Close()

	// Agregar el nuevo cliente al canal de audio
	audioChannel.mutex.Lock()
	audioChannel.clients[ws] = true
	audioChannel.mutex.Unlock()

	log.Println("Nuevo cliente conectado al canal de audio")

	for {
		messageType, message, err := ws.ReadMessage()
		if err != nil {
			log.Printf("Error al leer mensaje: %v", err)
			break
		}

		// Dependiendo del tipo de mensaje, realizamos diferentes acciones
		if messageType == websocket.TextMessage {
			switch string(message) {
			case "send":
				log.Println("Cliente enviando audio")
				// Mantener al cliente en modo de envío de audio
				err = handleSend(ws)
				if err != nil {
					log.Printf("Error al enviar audio: %v", err)
					break
				}
			case "listen":
				log.Println("Cliente escuchando audio")
				// Cliente está en modo escucha, no es necesario hacer nada específico aquí
			}
		}
	}

	// Remover el cliente del canal de audio al desconectarse
	audioChannel.mutex.Lock()
	delete(audioChannel.clients, ws)
	audioChannel.mutex.Unlock()

	log.Println("Cliente desconectado del canal de audio")
}

func handleSend(sender *websocket.Conn) error {
	for {
		// Leer mensaje binario (audio)
		_, audioData, err := sender.ReadMessage()
		if err != nil {
			log.Printf("Error al leer datos de audio: %v", err)
			return err
		}

		// Emitir el mensaje de audio a todos los demás clientes que están en modo "listen"
		err = broadcastAudioMessage(sender, audioData)
		if err != nil {
			log.Printf("Error al emitir mensaje de audio: %v", err)
			return err
		}
	}
}

func broadcastAudioMessage(sender *websocket.Conn, audioData []byte) error {
	audioChannel.mutex.Lock()
	defer audioChannel.mutex.Unlock()

	for client := range audioChannel.clients {
		if client != sender {
			err := client.WriteMessage(websocket.BinaryMessage, audioData)
			if err != nil {
				log.Printf("Error al enviar mensaje: %v", err)
				client.Close()
				delete(audioChannel.clients, client)
			}
		}
	}

	return nil
}

package client

import (
	"bytes"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Permitir todas las solicitudes CORS
	},
}

var audioClients = make(map[*websocket.Conn]bool)
var speechClients = make(map[*websocket.Conn]bool)
var mu sync.Mutex

// WebSocket para enviar audio en tiempo real
func HandleAudioConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error al actualizar la conexión: %v", err)
		return
	}
	defer ws.Close()

	mu.Lock()
	audioClients[ws] = true
	mu.Unlock()

	log.Println("Nuevo cliente conectado para envío de audio")

	var audioBuffer bytes.Buffer

	for {
		// Leer datos de audio del cliente
		_, audioData, err := ws.ReadMessage()
		if err != nil {
			log.Printf("Error al leer mensaje de audio: %v", err)
			break
		}

		// Transmitir el audio a todos los demás clientes conectados
		mu.Lock()
		for client := range audioClients {
			if client != ws {
				err := client.WriteMessage(websocket.BinaryMessage, audioData)
				if err != nil {
					log.Printf("Error al retransmitir audio: %v", err)
					client.Close()
					delete(audioClients, client)
				}
			}
		}
		mu.Unlock()

		// Enviar el audio al WebSocket de procesamiento para transcripción
		mu.Lock()
		for client := range speechClients {
			err := client.WriteMessage(websocket.BinaryMessage, audioData)
			if err != nil {
				log.Printf("Error al enviar audio para transcripción: %v", err)
				client.Close()
				delete(speechClients, client)
			}
		}
		mu.Unlock()

		// Acumular los datos de audio
		audioBuffer.Write(audioData)
	}

	mu.Lock()
	delete(audioClients, ws)
	mu.Unlock()

	log.Println("Cliente desconectado del envío de audio")
}

// WebSocket para procesar el audio y devolver texto transcrito
func HandleSpeechProcessing(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error al actualizar la conexión: %v", err)
		return
	}
	defer ws.Close()

	mu.Lock()
	speechClients[ws] = true
	mu.Unlock()

	log.Println("Nuevo cliente conectado para procesamiento de audio a texto")

	for {
		// Aquí se lee el audio enviado desde /ws, lo procesa y retorna el texto
		_, _, err := ws.ReadMessage()
		if err != nil {
			log.Printf("Cliente desconectado del procesamiento de audio a texto: %v", err)
			break
		}
	}

	mu.Lock()
	delete(speechClients, ws)
	mu.Unlock()
}

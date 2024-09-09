package client

import (
	"bytes"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/backsoul/walkie/internal/speech"
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

// WebSocket para enviar audio en tiempo real y procesar segmentos para convertirlos a texto
func HandleConnections(w http.ResponseWriter, r *http.Request) {
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
	ticker := time.NewTicker(2 * time.Second)

	go func() {
		for range ticker.C {
			if audioBuffer.Len() > 0 {
				audioData := audioBuffer.Bytes()
				if containsVoice(audioData) {
					go func(audio []byte) {
						text, err := speech.StreamAudioToText(bytes.NewReader(audio))
						if err != nil {
							log.Printf("Error al convertir audio a texto: %v", err)
							return
						}

						mu.Lock()
						for client := range speechClients {
							err := client.WriteMessage(websocket.TextMessage, []byte(text))
							if err != nil {
								log.Printf("Error al enviar texto: %v", err)
								client.Close()
								delete(speechClients, client)
							}
						}
						mu.Unlock()
					}(audioData)
				} else {
					log.Println("No se detectó voz en el audio.")
				}
				audioBuffer.Reset()
			}
		}
	}()

	for {
		_, audioData, err := ws.ReadMessage()
		if err != nil {
			log.Printf("Error al leer mensaje de audio: %v", err)
			break
		}

		mu.Lock()
		for client := range audioClients {
			if client != ws {
				err := client.WriteMessage(websocket.BinaryMessage, audioData)
				if err != nil {
					log.Printf("Error al enviar audio: %v", err)
					client.Close()
					delete(audioClients, client)
				}
			}
		}
		mu.Unlock()

		audioBuffer.Write(audioData)
	}

	mu.Lock()
	delete(audioClients, ws)
	mu.Unlock()

	log.Println("Cliente desconectado del envío de audio")
}

// WebSocket para recibir el texto transcrito
func HandleSpeechText(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error al actualizar la conexión: %v", err)
		return
	}
	defer ws.Close()

	mu.Lock()
	speechClients[ws] = true
	mu.Unlock()

	log.Println("Nuevo cliente conectado para recibir texto")

	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			log.Printf("Cliente desconectado de recepción de texto: %v", err)
			break
		}
	}

	mu.Lock()
	delete(speechClients, ws)
	mu.Unlock()
}

func containsVoice(audioData []byte) bool {
	var sum int64
	for _, b := range audioData {
		sum += int64(b)
	}
	avg := sum / int64(len(audioData))

	// Umbral para determinar si hay voz
	threshold := int64(50) // Puedes ajustar este valor según sea necesario
	return avg > threshold
}

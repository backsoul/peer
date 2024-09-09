package client

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/backsoul/walkie/internal/speech"
	"github.com/gorilla/websocket"
)

var (
	upgrader      = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	audioClients  = make(map[*websocket.Conn]chan []byte)
	speechClients = make(map[*websocket.Conn]bool)
	mu            sync.Mutex
)

// Buffer de audio y temporizador para fragmentar audio cada 5 segundos
var (
	audioBuffer      []byte
	fragmentDuration = 5 * time.Second
)

// WebSocket para enviar audio en tiempo real
func HandleAudioConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error al actualizar la conexión: %v", err)
		return
	}
	defer ws.Close()

	audioChan := make(chan []byte, 10)
	mu.Lock()
	audioClients[ws] = audioChan
	mu.Unlock()

	log.Println("Nuevo cliente conectado para envío de audio")

	// Gorutina para enviar audio a este cliente
	go func() {
		for audio := range audioChan {
			err := ws.WriteMessage(websocket.BinaryMessage, audio)
			if err != nil {
				log.Printf("Error al enviar audio: %v", err)
				ws.Close()
				break
			}
		}
	}()

	for {
		_, audioData, err := ws.ReadMessage()
		if err != nil {
			log.Printf("Error al leer mensaje de audio: %v", err)
			break
		}

		// Acumular audio en el buffer
		mu.Lock()
		audioBuffer = append(audioBuffer, audioData...)
		mu.Unlock()

		// Transmitir el audio a los demás clientes conectados
		go func(audio []byte) {
			mu.Lock()
			defer mu.Unlock()
			for client, ch := range audioClients {
				if client != ws {
					select {
					case ch <- audio:
					default:
						log.Printf("Cliente lento, descartando paquete de audio")
					}
				}
			}
		}(audioData)
	}

	mu.Lock()
	delete(audioClients, ws)
	close(audioChan)
	mu.Unlock()

	log.Println("Cliente desconectado del envío de audio")
}

// Fragmentar audio cada 5 segundos y procesarlo
func StartAudioFragmenter() {
	ticker := time.NewTicker(fragmentDuration)
	defer ticker.Stop()

	for range ticker.C {
		mu.Lock()
		if len(audioBuffer) > 0 {
			// Extraer y procesar el fragmento
			fragment := audioBuffer
			audioBuffer = nil // Reiniciar el buffer
			mu.Unlock()

			// Procesar el fragmento de audio
			go HandleSpeechProcessingAudio(fragment)
		} else {
			mu.Unlock()
		}
	}
}

// Procesar audio y convertirlo a texto
func HandleSpeechProcessingAudio(audioData []byte) {
	// Aquí se realiza la conversión del audio a texto
	text := speech.ConvertAudioToText(audioData) // Llamar al servicio de Google Speech-to-Text

	// Enviar el texto a los clientes conectados a /ws-speech
	mu.Lock()
	defer mu.Unlock()
	for client := range speechClients {
		err := client.WriteMessage(websocket.TextMessage, []byte(text))
		if err != nil {
			log.Printf("Error al enviar texto a cliente: %v", err)
			client.Close()
			delete(speechClients, client)
		}
	}
}

// WebSocket para recibir y procesar el audio a texto
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
		_, _, err := ws.ReadMessage()
		if err != nil {
			log.Printf("Error al leer mensaje en /ws-speech: %v", err)
			break
		}
	}

	mu.Lock()
	delete(speechClients, ws)
	mu.Unlock()

	log.Println("Cliente desconectado de procesamiento de audio a texto")
}

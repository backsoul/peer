package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"

	speech "cloud.google.com/go/speech/apiv1"
	"github.com/gorilla/websocket"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

// Definir un actualizador de WebSocket para manejar las conexiones
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Permitir todas las solicitudes CORS
		return true
	},
}

// Mapa para almacenar las conexiones de WebSocket
var clients = make(map[*websocket.Conn]bool)
var mu sync.Mutex // Mutex para evitar condiciones de carrera

func main() {
	http.HandleFunc("/ws", handleConnections)
	http.HandleFunc("/ws-speech", handleSpeechToText)

	port := ":3000"
	fmt.Printf("Servidor de WebSocket corriendo en http://localhost%s\n", port)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatalf("Error al iniciar el servidor: %v", err)
	}
}

// WebSocket para retransmitir audio
func handleConnections(w http.ResponseWriter, r *http.Request) {
	// Actualizar la conexión HTTP a una conexión WebSocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error al actualizar la conexión: %v", err)
		return
	}
	defer ws.Close()

	// Añadir el nuevo cliente al mapa
	mu.Lock()
	clients[ws] = true
	mu.Unlock()

	log.Println("Nuevo cliente conectado")

	for {
		// Leer el mensaje de audio desde el cliente
		_, message, err := ws.ReadMessage()
		if err != nil {
			log.Printf("Error al leer mensaje: %v", err)
			break
		}

		// Aquí puedes manejar el audio recibido
		log.Printf("Audio recibido: %d bytes", len(message))

		// Retransmitir el mensaje a todos los demás clientes conectados
		mu.Lock()
		for client := range clients {
			if client != ws {
				err := client.WriteMessage(websocket.BinaryMessage, message)
				if err != nil {
					log.Printf("Error al enviar mensaje a un cliente: %v", err)
					client.Close()
					delete(clients, client)
				}
			}
		}
		mu.Unlock()
	}

	// Eliminar al cliente cuando se desconecte
	mu.Lock()
	delete(clients, ws)
	mu.Unlock()

	log.Println("Cliente desconectado")
}

// WebSocket para convertir audio a texto
func handleSpeechToText(w http.ResponseWriter, r *http.Request) {
	// Actualizar la conexión HTTP a una conexión WebSocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error al actualizar la conexión: %v", err)
		return
	}
	defer ws.Close()

	log.Println("Cliente conectado para conversión de voz a texto")

	for {
		// Leer el mensaje de audio desde el cliente
		_, audioData, err := ws.ReadMessage()
		if err != nil {
			log.Printf("Error al leer mensaje de audio: %v", err)
			break
		}

		// Convertir el audio a texto
		text, err := convertSpeechToText(audioData)
		if err != nil {
			log.Printf("Error al convertir audio a texto: %v", err)
			continue
		}

		// Enviar el texto de vuelta al cliente
		err = ws.WriteMessage(websocket.TextMessage, []byte(text))
		if err != nil {
			log.Printf("Error al enviar mensaje de texto: %v", err)
			break
		}
	}

	log.Println("Cliente desconectado de conversión de voz a texto")
}

// Función para convertir audio a texto usando Google Speech-to-Text
func convertSpeechToText(audioData []byte) (string, error) {
	ctx := context.Background()
	client, err := speech.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("error al crear el cliente de Google Cloud Speech: %v", err)
	}
	defer client.Close()

	req := &speechpb.RecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:        speechpb.RecognitionConfig_LINEAR16,
			SampleRateHertz: 16000,
			LanguageCode:    "es-ES", // Cambia esto al idioma que necesites
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{
				Content: audioData,
			},
		},
	}

	resp, err := client.Recognize(ctx, req)
	if err != nil {
		return "", fmt.Errorf("error al procesar el audio: %v", err)
	}

	if len(resp.Results) == 0 {
		return "", fmt.Errorf("no se encontraron resultados")
	}

	return resp.Results[0].Alternatives[0].Transcript, nil
}

package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

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

// Mapa para almacenar las conexiones de WebSocket de los clientes que envían y reciben audio
var audioClients = make(map[*websocket.Conn]bool)

// Mapa para almacenar las conexiones de WebSocket de los clientes que reciben texto
var speechClients = make(map[*websocket.Conn]bool)

var mu sync.Mutex // Mutex para evitar condiciones de carrera

func main() {
	http.HandleFunc("/ws", handleConnections)       // WebSocket para enviar audio y retransmitirlo
	http.HandleFunc("/ws-speech", handleSpeechText) // WebSocket para recibir texto

	port := ":3000"
	fmt.Printf("Servidor de WebSocket corriendo en http://localhost%s\n", port)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatalf("Error al iniciar el servidor: %v", err)
	}
}

// Función para verificar si el audio contiene voz (si hay amplitud suficiente)
func containsVoice(audioData []byte) bool {
	// Calcular la amplitud promedio del audio
	var sum int64
	for _, b := range audioData {
		sum += int64(b)
	}
	avg := sum / int64(len(audioData))

	// Establecer un umbral de amplitud para considerar que hay "voz"
	threshold := int64(50) // Puedes ajustar este valor
	return avg > threshold
}

// WebSocket para enviar audio en tiempo real y procesar segmentos de 5 segundos para convertirlos a texto
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
	audioClients[ws] = true
	mu.Unlock()

	log.Println("Nuevo cliente conectado para envío de audio")

	var audioBuffer bytes.Buffer
	ticker := time.NewTicker(5 * time.Second)

	go func() {
		for range ticker.C {
			if audioBuffer.Len() > 0 {
				audioData := audioBuffer.Bytes()

				// Verificar si el audio tiene suficiente actividad de voz antes de enviarlo a Google
				if containsVoice(audioData) {
					// Convertir el audio acumulado a texto
					go func(audio []byte) {
						text, err := convertSpeechToText(audio)
						if err != nil {
							log.Printf("Error al convertir audio a texto: %v", err)
							return
						}

						// Enviar el texto a los clientes conectados a /ws-speech
						mu.Lock()
						for client := range speechClients {
							err := client.WriteMessage(websocket.TextMessage, []byte(text))
							if err != nil {
								log.Printf("Error al enviar texto a un cliente: %v", err)
								client.Close()
								delete(speechClients, client)
							}
						}
						mu.Unlock()
					}(audioData)
				} else {
					log.Println("No se detectó voz en el audio.")
				}

				// Limpiar el buffer para el siguiente periodo de 5 segundos
				audioBuffer.Reset()
			}
		}
	}()

	for {
		// Leer el mensaje de audio desde el cliente
		_, audioData, err := ws.ReadMessage()
		if err != nil {
			log.Printf("Error al leer mensaje de audio: %v", err)
			break
		}

		// Enviar el audio en tiempo real a los demás clientes conectados a /ws
		mu.Lock()
		for client := range audioClients {
			if client != ws { // Evitar enviar el audio de vuelta al mismo cliente
				err := client.WriteMessage(websocket.BinaryMessage, audioData)
				if err != nil {
					log.Printf("Error al enviar audio a un cliente: %v", err)
					client.Close()
					delete(audioClients, client)
				}
			}
		}
		mu.Unlock()

		// Acumular el audio recibido en el buffer para la transcripción
		audioBuffer.Write(audioData)
	}

	// Eliminar al cliente cuando se desconecte
	mu.Lock()
	delete(audioClients, ws)
	mu.Unlock()

	log.Println("Cliente desconectado del envío de audio")
}

// WebSocket para recibir el texto transcrito
func handleSpeechText(w http.ResponseWriter, r *http.Request) {
	// Actualizar la conexión HTTP a una conexión WebSocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error al actualizar la conexión: %v", err)
		return
	}
	defer ws.Close()

	// Añadir el nuevo cliente al mapa
	mu.Lock()
	speechClients[ws] = true
	mu.Unlock()

	log.Println("Nuevo cliente conectado para recibir texto")

	// Mantener la conexión abierta hasta que el cliente se desconecte
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			log.Printf("Cliente desconectado de recepción de texto: %v", err)
			break
		}
	}

	// Eliminar al cliente cuando se desconecte
	mu.Lock()
	delete(speechClients, ws)
	mu.Unlock()
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
			SampleRateHertz: 44100,
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

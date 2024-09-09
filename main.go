package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/backsoul/walkie/internal/client"
)

func main() {
	// WebSocket para enviar audio y retransmitirlo
	http.HandleFunc("/ws", client.HandleAudioConnections)

	// WebSocket para recibir y procesar el audio a texto
	// http.HandleFunc("/ws-speech", client.HandleSpeechProcessing)

	port := ":3000"
	fmt.Printf("Servidor de WebSocket corriendo en http://localhost%s\n", port)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatalf("Error al iniciar el servidor: %v", err)
	}
}

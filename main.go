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

	// Limpieza de mensajes pendientes cada 10 segundos
	// go client.CleanUpOldMessages(10 * time.Second)

	port := ":3000"
	fmt.Printf("Servidor de WebSocket corriendo en https://localhost%s\n", port)

	// Servir usando TLS
	err := http.ListenAndServeTLS(":3000", "localhost.crt", "localhost.key", nil)
	if err != nil {
		log.Fatalf("Error al iniciar el servidor: %v", err)
	}
}

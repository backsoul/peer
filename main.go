package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// Definir un actualizador de WebSocket para manejar las conexiones
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Permitir todas las solicitudes CORS
		return true
	},
}

func main() {
	http.HandleFunc("/ws", handleConnections)

	port := ":3000"
	fmt.Printf("Servidor de WebSocket corriendo en http://localhost%s\n", port)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatalf("Error al iniciar el servidor: %v", err)
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	// Actualizar la conexión HTTP a una conexión WebSocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error al actualizar la conexión: %v", err)
		return
	}
	defer ws.Close()

	log.Println("Nuevo cliente conectado")

	for {
		// Leer el mensaje de audio desde el cliente
		_, message, err := ws.ReadMessage()
		if err != nil {
			log.Printf("Error al leer mensaje: %v", err)
			break
		}

		// Aquí puedes manejar el audio recibido, por ejemplo, retransmitirlo a otros clientes
		log.Printf("Audio recibido: %d bytes", len(message))

		// Enviar una respuesta al cliente si es necesario
		err = ws.WriteMessage(websocket.BinaryMessage, message)
		if err != nil {
			log.Printf("Error al enviar mensaje: %v", err)
			break
		}
	}

	log.Println("Cliente desconectado")
}

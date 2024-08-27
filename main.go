package main

import (
	"fmt"
	"log"
	"net/http"

	socketio "github.com/googollee/go-socket.io"
	"github.com/rs/cors"
)

func main() {
	// Crear un servidor de Socket.IO
	server := socketio.NewServer(nil)

	// Manejar el evento "connection"
	server.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		fmt.Println("Nuevo cliente conectado:", s.ID())
		return nil
	})

	// Manejar el evento "audio"
	server.OnEvent("/", "audio", func(s socketio.Conn, audioData []byte) {
		// Aquí manejas los datos de audio recibidos
		fmt.Println("Audio recibido del cliente:", s.ID())
		// Opcional: Procesar el audio o retransmitirlo a otros clientes
		server.BroadcastToNamespace("/", "audio", audioData)
	})

	// Manejar el evento "disconnect"
	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		fmt.Println("Cliente desconectado:", s.ID(), "Razón:", reason)
	})

	// Iniciar el servidor de Socket.IO en un goroutine
	go server.Serve()
	defer server.Close()

	// Crear un mux HTTP
	mux := http.NewServeMux()

	// Ruta para manejar el servidor de Socket.IO
	mux.Handle("/socket.io/", server)

	// Permitir todas las solicitudes CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, // Permitir todas las solicitudes
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Origin", "Content-Type", "Accept"},
	})

	// Crear un servidor HTTP con CORS habilitado
	handler := c.Handler(mux)

	port := ":3000"
	fmt.Printf("Servidor de audio en tiempo real corriendo en http://localhost%s\n", port)
	err := http.ListenAndServe(port, handler)
	if err != nil {
		log.Fatalf("Error al iniciar el servidor: %v", err)
	}
}

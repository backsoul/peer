package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/backsoul/walkie/internal/client"
)

func main() {
	// Ruta para manejar la negociación de WebRTC
	http.HandleFunc("/offer", client.HandleOffer)

	port := ":3000"
	fmt.Printf("Servidor WebRTC corriendo en https://localhost%s\n", port)

	// Servir usando TLS
	// err := http.ListenAndServeTLS(":3000", "localhost.crt", "localhost.key", nil)
	// if err != nil {
	// 	log.Fatalf("Error al iniciar el servidor: %v", err)
	// }
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatalf("Error al iniciar el servidor: %v", err)
	}
}

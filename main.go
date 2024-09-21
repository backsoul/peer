package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/backsoul/walkie/internal/routes"
)

// Mutex para proteger el acceso a rooms

func main() {
	http.HandleFunc("/", routes.HandleConnection)

	port := "3000"
	fmt.Printf("Server listening on port %s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

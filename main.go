package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/backsoul/walkie/internal/routes"
)

func main() {
	// Get the local machine's IP address (in the local network)
	localIP, err := getLocalIP()
	if err != nil {
		localIP = "127.0.0.1"
		fmt.Println("Error getting local IP address:", err)
	}

	// Define the folder where your Angular app is built (adjust the path to where your 'dist' folder is located)
	angularDistFolder := "./public" // Adjust this path if needed

	// Serve static files (your Angular app) from the 'public' folder
	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir(angularDistFolder))))

	// Handle WebSocket connections under a separate route
	http.HandleFunc("/ws", routes.HandleConnection)

	// Print the server URL with local network IP
	fmt.Printf("Server listening on https://%s:3000\n", localIP)

	// Create the server with TLS configuration
	server := &http.Server{
		Addr: ":3000", // HTTPS port
		TLSConfig: &tls.Config{
			InsecureSkipVerify: false, // Be sure to set this to false for production
		},
	}

	// Serve the app with the specified SSL certificate and private key
	err = server.ListenAndServeTLS("localhost.crt", "localhost.key")
	if err != nil {
		log.Fatal("ListenAndServeTLS: ", err)
	}
}

// getLocalIP retrieves the local machine's IP address.
func getLocalIP() (string, error) {
	// Get a list of network interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	// Loop through interfaces and check for an active one
	for _, iface := range interfaces {
		// Skip interfaces that are down or virtual interfaces like lo0
		if iface.Flags&net.FlagUp == 0 || iface.Name == "lo0" {
			continue
		}

		// Get a list of addresses for the interface
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}

		// Check each address for an IPv4 address (not IPv6)
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.To4() == nil {
				continue
			}

			// Return the first found IPv4 address
			return ipNet.IP.String(), nil
		}
	}

	return "", fmt.Errorf("local IP not found")
}

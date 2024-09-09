package client

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/pion/webrtc/v3"
)

var (
	peerConnections = make(map[string]*webrtc.PeerConnection) // Almacena las conexiones WebRTC de los clientes
	mu              sync.Mutex                                // Para sincronizar el acceso a las conexiones
)

// broadcastMessage estructura para manejar la retransmisión
type broadcastMessage struct {
	Data   []byte
	Sender *webrtc.PeerConnection
}

// Canal para manejar la retransmisión de audio
var broadcastCh = make(chan broadcastMessage, 100)

// Inicializar la retransmisión en un goroutine separado
func init() {
	go handleBroadcasts()
}

// Manejamos las ofertas de conexión WebRTC
func HandleOffer(w http.ResponseWriter, r *http.Request) {
	// Obtener la oferta SDP del cliente
	var offer webrtc.SessionDescription
	err := json.NewDecoder(r.Body).Decode(&offer)
	if err != nil {
		http.Error(w, "Invalid offer", http.StatusBadRequest)
		return
	}

	// Crear una nueva conexión Peer WebRTC
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		log.Fatalf("Error creando PeerConnection: %v", err)
	}

	// Canal de datos para enviar y recibir audio
	peerConnection.OnDataChannel(func(dc *webrtc.DataChannel) {
		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			// Enviar el mensaje al canal de retransmisión
			broadcastCh <- broadcastMessage{
				Data:   msg.Data,
				Sender: peerConnection,
			}
		})
	})

	// Almacenar la conexión en el map de conexiones
	mu.Lock()
	peerConnections[peerConnection.RemoteDescription().SDP] = peerConnection
	mu.Unlock()

	// Establecer el SDP remoto
	err = peerConnection.SetRemoteDescription(offer)
	if err != nil {
		log.Fatalf("Error al establecer la descripción remota: %v", err)
	}

	// Crear la respuesta SDP
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		log.Fatalf("Error creando respuesta: %v", err)
	}

	// Establecer la descripción local
	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		log.Fatalf("Error al establecer la descripción local: %v", err)
	}

	// Enviar la respuesta SDP de vuelta al cliente
	response := peerConnection.LocalDescription()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Maneja la retransmisión de audio a todos los clientes conectados
func handleBroadcasts() {
	for msg := range broadcastCh {
		retransmitAudio(msg.Data, msg.Sender)
	}
}

// Retransmite el audio a todos los clientes conectados, excepto al remitente
func retransmitAudio(data []byte, sender *webrtc.PeerConnection) {
	mu.Lock()
	defer mu.Unlock()

	// Iterar sobre todas las conexiones activas
	for _, client := range peerConnections {
		if client != sender { // No enviar el audio de vuelta al remitente
			// Crear un canal de datos o usar uno existente
			dataChannel, err := client.CreateDataChannel("audio", nil)
			if err != nil {
				log.Printf("Error creando DataChannel: %v", err)
				continue
			}

			// Enviar el audio por el DataChannel
			err = dataChannel.Send(data)
			if err != nil {
				log.Printf("Error enviando el audio: %v", err)
			} else {
				fmt.Println("Audio retransmitido a un cliente")
			}
		}
	}
}

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/pion/webrtc/v3"
)

var (
	// Mapa para almacenar las conexiones de WebRTC usando un identificador único (por ejemplo, ID de sesión)
	peerConnections = make(map[string]*webrtc.PeerConnection)
	peerLock        sync.Mutex
)

// Estructura para manejar los mensajes de ICE candidate
type IceCandidate struct {
	Candidate     string `json:"candidate"`
	SDPMid        string `json:"sdpMid"`
	SDPMLineIndex uint16 `json:"sdpMLineIndex"`
}

// Habilitar CORS
func enableCORS(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type")
	(*w).Header().Set("Access-Control-Max-Age", "86400") // Cache the preflight response
}

// Manejar ofertas de WebRTC
func handleOffer(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Println("Solicitud recibida en /offer")

	// Decodificar la oferta
	offer := webrtc.SessionDescription{}
	if err := json.NewDecoder(r.Body).Decode(&offer); err != nil {
		log.Printf("Error decoding offer: %v", err)
		http.Error(w, "Failed to decode offer", http.StatusBadRequest)
		return
	}

	// Crear una nueva conexión PeerConnection
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		log.Printf("Error creating PeerConnection: %v", err)
		http.Error(w, "Failed to create PeerConnection", http.StatusInternalServerError)
		return
	}

	// Bloquear el acceso a la lista de conexiones para agregar la nueva
	peerLock.Lock()
	sessionID := "session_id" // Aquí puedes usar un identificador único (por ejemplo, token de sesión)
	peerConnections[sessionID] = peerConnection
	peerLock.Unlock()

	// Establecer la descripción remota con la oferta
	if err := peerConnection.SetRemoteDescription(offer); err != nil {
		log.Printf("Error setting remote description: %v", err)
		http.Error(w, "Failed to set remote description", http.StatusInternalServerError)
		return
	}

	// Crear una respuesta (answer)
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		log.Printf("Error creating answer: %v", err)
		http.Error(w, "Failed to create answer", http.StatusInternalServerError)
		return
	}

	// Establecer la descripción local con la respuesta
	if err := peerConnection.SetLocalDescription(answer); err != nil {
		log.Printf("Error setting local description: %v", err)
		http.Error(w, "Failed to set local description", http.StatusInternalServerError)
		return
	}

	// Enviar la respuesta al cliente
	if err := json.NewEncoder(w).Encode(peerConnection.LocalDescription()); err != nil {
		log.Printf("Error encoding answer: %v", err)
		http.Error(w, "Failed to encode answer", http.StatusInternalServerError)
		return
	}

	// Manejar candidatos de ICE que se generen
	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}

		// Aquí puedes enviar el candidato al cliente
		log.Printf("Nuevo candidato ICE generado: %v", candidate.ToJSON())
	})
}

// Manejar candidatos de ICE del cliente
func handleICECandidate(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Println("Solicitud recibida en /ice-candidate")

	var iceCandidate IceCandidate
	if err := json.NewDecoder(r.Body).Decode(&iceCandidate); err != nil {
		log.Printf("Error decoding ICE candidate: %v", err)
		http.Error(w, "Failed to decode ICE candidate", http.StatusBadRequest)
		return
	}

	// Bloquear el acceso a la lista de conexiones
	peerLock.Lock()
	peerConnection := peerConnections["session_id"] // Aquí se debe usar el mismo ID de sesión que en el offer
	peerLock.Unlock()

	if peerConnection == nil {
		log.Println("No se encontró la conexión para la sesión")
		http.Error(w, "PeerConnection not found", http.StatusNotFound)
		return
	}

	// Agregar el candidato de ICE a la conexión
	candidate := webrtc.ICECandidateInit{
		Candidate:     iceCandidate.Candidate,
		SDPMid:        &iceCandidate.SDPMid,
		SDPMLineIndex: &iceCandidate.SDPMLineIndex,
	}
	if err := peerConnection.AddICECandidate(candidate); err != nil {
		log.Printf("Error adding ICE candidate: %v", err)
		http.Error(w, "Failed to add ICE candidate", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func main() {
	// Ruta para manejar la oferta SDP
	http.HandleFunc("/offer", handleOffer)

	// Ruta para manejar los candidatos de ICE
	http.HandleFunc("/ice-candidate", handleICECandidate)

	log.Println("Servidor WebRTC escuchando en :3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}

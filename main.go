package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/pion/webrtc/v3"
)

func enableCORS(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func main() {
	http.HandleFunc("/offer", func(w http.ResponseWriter, r *http.Request) {
		enableCORS(&w)

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		log.Println("Solicitud recibida en /offer")

		offer := webrtc.SessionDescription{}
		if err := json.NewDecoder(r.Body).Decode(&offer); err != nil {
			log.Printf("Error decoding offer: %v", err)
			http.Error(w, "Failed to decode offer", http.StatusBadRequest)
			return
		}

		peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
		if err != nil {
			log.Printf("Error creating PeerConnection: %v", err)
			http.Error(w, "Failed to create PeerConnection", http.StatusInternalServerError)
			return
		}

		if err := peerConnection.SetRemoteDescription(offer); err != nil {
			log.Printf("Error setting remote description: %v", err)
			http.Error(w, "Failed to set remote description", http.StatusInternalServerError)
			return
		}

		answer, err := peerConnection.CreateAnswer(nil)
		if err != nil {
			log.Printf("Error creating answer: %v", err)
			http.Error(w, "Failed to create answer", http.StatusInternalServerError)
			return
		}

		if err := peerConnection.SetLocalDescription(answer); err != nil {
			log.Printf("Error setting local description: %v", err)
			http.Error(w, "Failed to set local description", http.StatusInternalServerError)
			return
		}

		if err := json.NewEncoder(w).Encode(peerConnection.LocalDescription()); err != nil {
			log.Printf("Error encoding answer: %v", err)
			http.Error(w, "Failed to encode answer", http.StatusInternalServerError)
			return
		}
	})

	log.Println("Servidor WebRTC escuchando en :3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}

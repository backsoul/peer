package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var rooms = make(map[string]map[*websocket.Conn]string) // map connection to UUID

func main() {
	http.HandleFunc("/", handleConnection)

	port := "3000"
	fmt.Printf("Server listening on port %s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func handleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error while upgrading connection:", err)
		return
	}
	defer conn.Close()

	clientUUID := uuid.New().String() // Generate a UUID for the client
	fmt.Println("Client connected with UUID:", clientUUID)

	// Send the UUID to the client
	conn.WriteMessage(websocket.TextMessage, encodeJSON(map[string]interface{}{
		"type": "uuid",
		"uuid": clientUUID,
	}))

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error while reading message:", err)
			break
		}

		var data map[string]interface{}
		if err := json.Unmarshal(message, &data); err != nil {
			log.Println("Error unmarshaling message:", err)
			continue
		}

		messageType, ok := data["type"].(string)
		if !ok {
			log.Println("Invalid message format")
			continue
		}

		switch messageType {
		case "join":
			handleJoin(conn, data, clientUUID)
		case "start_call", "webrtc_offer", "webrtc_answer", "webrtc_ice_candidate":
			handleRoomMessage(data, conn, clientUUID)
		default:
			log.Printf("Unknown message type: %s\n", messageType)
		}
	}
}

func handleJoin(conn *websocket.Conn, data map[string]interface{}, clientUUID string) {
	roomID, _ := data["roomId"].(string)
	room, exists := rooms[roomID]
	if !exists {
		room = make(map[*websocket.Conn]string)
		rooms[roomID] = room
	}

	room[conn] = clientUUID

	if len(room) == 1 {
		conn.WriteMessage(websocket.TextMessage, encodeJSON(map[string]interface{}{
			"type":   "room_created",
			"roomId": roomID,
		}))
	} else {
		conn.WriteMessage(websocket.TextMessage, encodeJSON(map[string]interface{}{
			"type":   "room_joined",
			"roomId": roomID,
		}))
		broadcast(roomID, map[string]interface{}{
			"type": "start_call",
		}, clientUUID)
	}
}

func handleRoomMessage(data map[string]interface{}, conn *websocket.Conn, clientUUID string) {
	roomID, _ := data["roomId"].(string)
	room, exists := rooms[roomID]
	if exists {
		for client, _ := range room {
			if client != conn {
				message := data
				message["from"] = clientUUID // Add the sender's UUID
				client.WriteMessage(websocket.TextMessage, encodeJSON(message))
			}
		}
	}
}

func broadcast(roomID string, message map[string]interface{}, clientUUID string) {
	room, exists := rooms[roomID]
	if exists {
		for client, uuid := range room {
			if uuid != clientUUID {
				message["from"] = clientUUID // Include the sender's UUID
				client.WriteMessage(websocket.TextMessage, encodeJSON(message))
			}
		}
	}
}

func encodeJSON(data map[string]interface{}) []byte {
	message, err := json.Marshal(data)
	if err != nil {
		log.Println("Error marshaling JSON:", err)
		return nil
	}
	return message
}

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var rooms = make(map[string]map[*websocket.Conn]bool)

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

	fmt.Println("Client connected")

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
			handleJoin(conn, data)
		case "start_call":
			handleRoomMessage(data, conn)
		case "webrtc_offer":
			handleRoomMessage(data, conn)
		case "webrtc_answer":
			handleRoomMessage(data, conn)
		case "webrtc_ice_candidate":
			handleRoomMessage(data, conn)
		default:
			log.Printf("Unknown message type: %s\n", messageType)
		}
	}
}

func handleJoin(conn *websocket.Conn, data map[string]interface{}) {
	roomID, _ := data["roomId"].(string)
	room, exists := rooms[roomID]
	if !exists {
		room = make(map[*websocket.Conn]bool)
		rooms[roomID] = room
	}

	// if len(room) >= 2 {
	// 	conn.WriteMessage(websocket.TextMessage, encodeJSON(map[string]interface{}{
	// 		"type":   "full_room",
	// 		"roomId": roomID,
	// 	}))
	// 	return
	// }

	room[conn] = true

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
		})
	}
}

func handleRoomMessage(data map[string]interface{}, conn *websocket.Conn) {
	roomID, _ := data["roomId"].(string)
	room, exists := rooms[roomID]
	if exists {
		for client := range room {
			if client != conn {
				client.WriteMessage(websocket.TextMessage, encodeJSON(data))
			}
		}
	}
}

func broadcast(roomID string, message map[string]interface{}) {
	room, exists := rooms[roomID]
	if exists {
		for client := range room {
			client.WriteMessage(websocket.TextMessage, encodeJSON(message))
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

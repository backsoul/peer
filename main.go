package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var rooms = make(map[string]map[*websocket.Conn]string) // map connection to UUID
var mu sync.Mutex                                       // Mutex para proteger el acceso a rooms

type Connection struct {
	conn       *websocket.Conn
	send       chan []byte
	clientUUID string
}

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

	clientUUID := uuid.New().String() // Genera un UUID para el cliente
	fmt.Println("Client connected with UUID:", clientUUID)

	// Crea la estructura Connection con un canal de escritura
	connection := &Connection{
		conn:       conn,
		send:       make(chan []byte, 256), // Canal bufferizado para escribir mensajes
		clientUUID: clientUUID,
	}

	// Lanzar una goroutine que escuche el canal de send y escriba en el WebSocket
	go handleWrites(connection)

	// Enviar el UUID al cliente
	connection.send <- encodeJSON(map[string]interface{}{
		"type": "uuid",
		"uuid": clientUUID,
	})

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
			handleJoin(connection, data)
		case "start_call", "webrtc_offer", "webrtc_answer", "webrtc_ice_candidate":
			handleRoomMessage(data, connection)
		case "close 1001 (going away)":
			log.Printf("client disconnected: %s\n", data)
		default:
			log.Printf("Unknown message type: %s\n", messageType)
		}
	}
}

func handleWrites(c *Connection) {
	for msg := range c.send {
		err := c.conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Println("Error writing message:", err)
			break
		}
	}
}

func handleJoin(connection *Connection, data map[string]interface{}) {
	roomID, _ := data["roomId"].(string)

	mu.Lock() // Bloquear el acceso concurrente a rooms
	room, exists := rooms[roomID]
	if !exists {
		room = make(map[*websocket.Conn]string)
		rooms[roomID] = room
	}
	room[connection.conn] = connection.clientUUID
	mu.Unlock() // Desbloquear

	if len(room) == 1 {
		connection.send <- encodeJSON(map[string]interface{}{
			"type":   "room_created",
			"roomId": roomID,
		})
	} else {
		connection.send <- encodeJSON(map[string]interface{}{
			"type":   "room_joined",
			"roomId": roomID,
		})
		broadcast(roomID, map[string]interface{}{
			"type": "start_call",
		}, connection.clientUUID)
	}
}

func handleRoomMessage(data map[string]interface{}, connection *Connection) {
	roomID, _ := data["roomId"].(string)
	mu.Lock()
	room, exists := rooms[roomID]
	mu.Unlock()

	if exists {
		for client, _ := range room {
			if client != connection.conn {
				message := data
				message["from"] = connection.clientUUID // Añadir el UUID del remitente
				rooms[roomID][client] = connection.clientUUID
				client.WriteMessage(websocket.TextMessage, encodeJSON(message))
			}
		}
	}
}

func broadcast(roomID string, message map[string]interface{}, clientUUID string) {
	mu.Lock()
	room, exists := rooms[roomID]
	mu.Unlock()

	if exists {
		for client, uuid := range room {
			if uuid != clientUUID {
				message["from"] = clientUUID // Incluir el UUID del remitente
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

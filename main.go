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

var rooms = make(map[string]map[*websocket.Conn]*Connection) // map connection to Connection struct
var mu sync.Mutex                                            // Mutex para proteger el acceso a rooms

type Connection struct {
	conn       *websocket.Conn
	send       chan []byte
	clientUUID string
	cameraOn   bool // Estado de la cámara
	audioOn    bool // Estado del micrófono
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

	connection := &Connection{
		conn:       conn,
		send:       make(chan []byte, 256),
		clientUUID: clientUUID,
		cameraOn:   false, // Inicializar en false
		audioOn:    false, // Inicializar en false
	}

	go handleWrites(connection)

	// Enviar el UUID al cliente
	connection.send <- encodeJSON(map[string]interface{}{
		"type": "uuid",
		"uuid": clientUUID,
	})

	var roomID string

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("Client disconnected: %s\n", clientUUID)
				handleClientDisconnect(roomID, connection)
				break
			}
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
			roomID = handleJoin(connection, data)
		case "close_call":
			handleClientDisconnect(roomID, connection)
		case "start_call", "webrtc_offer", "webrtc_answer", "webrtc_ice_candidate":
			handleRoomMessage(data, connection)
		case "mic_on_remote", "mic_off_remote", "video_on_remote", "video_off_remote":
			handleDevicesStatus(roomID, connection, data)
		default:
			log.Printf("Unknown message type: %s\n", messageType)
		}
	}
}

func handleDevicesStatus(roomID string, connection *Connection, data map[string]interface{}) {
	mu.Lock()
	defer mu.Unlock()

	// Actualizar el estado de la cámara y micrófono según el mensaje recibido
	switch data["type"].(string) {
	case "video_on_remote":
		connection.cameraOn = true
	case "video_off_remote":
		connection.cameraOn = false
	case "mic_on_remote":
		connection.audioOn = true
	case "mic_off_remote":
		connection.audioOn = false
	}

	// Difundir el nuevo estado a los otros clientes de la sala
	broadcast(roomID, map[string]interface{}{
		"type":     data["type"],
		"uuid":     connection.clientUUID,
		"cameraOn": connection.cameraOn,
		"audioOn":  connection.audioOn,
	}, connection.clientUUID)
}

func handleJoin(connection *Connection, data map[string]interface{}) string {
	roomID := data["roomId"].(string)

	mu.Lock()
	defer mu.Unlock()

	room, exists := rooms[roomID]
	if !exists {
		room = make(map[*websocket.Conn]*Connection)
		rooms[roomID] = room
	}
	room[connection.conn] = connection

	// Notificar si la sala fue creada o si se unió
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
		// Notificar a los demás clientes que la llamada ha comenzado
		broadcast(roomID, map[string]interface{}{
			"type": "start_call",
		}, connection.clientUUID)
	}

	return roomID
}

func handleClientDisconnect(roomID string, connection *Connection) {
	mu.Lock()
	defer mu.Unlock()

	room, exists := rooms[roomID]
	if exists {
		delete(room, connection.conn)

		// Notificar a los otros clientes sobre la desconexión
		if len(room) > 0 {
			broadcast(roomID, map[string]interface{}{
				"type":   "client_disconnected",
				"uuid":   connection.clientUUID,
				"roomId": roomID,
			}, connection.clientUUID)
		} else {
			delete(rooms, roomID)
		}
	}
}

func handleRoomMessage(data map[string]interface{}, connection *Connection) {
	roomID := data["roomId"].(string)

	mu.Lock()
	room, exists := rooms[roomID]
	mu.Unlock()

	if exists {
		for client, _ := range room {
			if client != connection.conn {
				data["from"] = connection.clientUUID
				client.WriteMessage(websocket.TextMessage, encodeJSON(data))
			}
		}
	}
}

func broadcast(roomID string, message map[string]interface{}, senderUUID string) {
	mu.Lock()
	defer mu.Unlock()

	room, exists := rooms[roomID]
	if !exists {
		return
	}

	for client, conn := range room {
		if conn.clientUUID == senderUUID {
			continue
		}
		err := client.WriteMessage(websocket.TextMessage, encodeJSON(message))
		if err != nil {
			client.Close()
			delete(room, client)
		}
	}

	// Si la room está vacía, eliminarla
	if len(room) == 0 {
		delete(rooms, roomID)
	}
}

func handleWrites(c *Connection) {
	defer func() {
		close(c.send)
	}()

	for msg := range c.send {
		err := c.conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			break
		}
	}
}

func encodeJSON(data map[string]interface{}) []byte {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Println("Error encoding JSON:", err)
		return nil
	}
	return jsonData
}

package routes

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Configuración para la conexión WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Constantes de configuración
const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

// Variables globales
var (
	rooms = make(map[string]map[*websocket.Conn]string)
	mu    sync.Mutex
)

// Connection representa una conexión WebSocket
type Connection struct {
	conn           *websocket.Conn
	send           chan []byte
	clientUUID     string
	cameraOn       bool
	audioOn        bool
	maxMessageSize int64
}

// HandleConnection gestiona la nueva conexión WebSocket
func HandleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading connection: %v", err)
		return
	}
	defer conn.Close()

	clientUUID := uuid.New().String()

	// Leer el mensaje inicial para configuración
	data, err := readClientMessage(conn)
	if err != nil {
		log.Printf("Error reading initial config: %v", err)
		return
	}

	// Establecer tamaño máximo de mensaje, si es proporcionado por el cliente
	var maxMessageSize int64 = 4096
	if size, ok := data["maxMessageSize"].(float64); ok {
		maxMessageSize = int64(size)
	}

	connection := initializeConnection(conn, clientUUID, maxMessageSize)

	conn.SetReadLimit(connection.maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	go handleWrites(connection)
	sendUUIDToClient(connection, clientUUID)

	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ticker.C:
				conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Printf("Error sending ping: %v", err)
					return
				}
			}
		}
	}()

	handleClientMessages(conn, connection)
}

func initializeConnection(conn *websocket.Conn, clientUUID string, maxMsgSize int64) *Connection {
	return &Connection{
		conn:           conn,
		send:           make(chan []byte, 512),
		clientUUID:     clientUUID,
		maxMessageSize: maxMsgSize,
	}
}

func sendUUIDToClient(connection *Connection, clientUUID string) {
	connection.send <- encodeJSON(map[string]interface{}{
		"type": "uuid",
		"uuid": clientUUID,
	})
}

func handleWrites(connection *Connection) {
	defer close(connection.send)
	for msg := range connection.send {
		connection.conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := connection.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Printf("Error writing message: %v", err)
			break
		}
	}
}

func handleClientMessages(conn *websocket.Conn, connection *Connection) {
	var roomID string
	for {
		data, err := readClientMessage(conn) // messageType no es necesario aquí
		if err != nil {
			handleClientError(connection, roomID, err)
			break
		}

		switch data["type"] {
		case "join":
			roomID = handleJoin(connection, data)
		case "close_call":
			handleClientDisconnect(roomID, connection)
		case "video_on_remote", "video_off_remote", "mic_on_remote", "mic_off_remote":
			handleDevicesStatus(connection, data)
		case "transcript":
			handleTranscript(data, connection)
		default:
			handleRoomMessage(data, connection)
		}
	}
}

func readClientMessage(conn *websocket.Conn) (map[string]interface{}, error) {
	_, message, err := conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(message, &data); err != nil {
		return nil, fmt.Errorf("error unmarshaling message: %v", err)
	}

	return data, nil // Eliminamos `messageType` ya que se puede acceder a él desde `data`
}

func handleClientError(connection *Connection, roomID string, err error) {
	if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
		log.Printf("Client disconnected: %s", connection.clientUUID)
		handleClientDisconnect(roomID, connection)
	} else {
		log.Printf("Error reading message: %v", err)
		handleClientDisconnect(roomID, connection)
	}
}

func handleJoin(connection *Connection, data map[string]interface{}) string {
	roomID := getRoomID(data)

	room := addToRoom(connection, roomID)

	if len(room) == 1 {
		notifyClient(connection, "room_created", roomID)
	} else {
		notifyClient(connection, "room_joined", roomID)
		broadcastToRoom(roomID, map[string]interface{}{"type": "start_call"}, connection.clientUUID)
	}

	return roomID
}

func getRoomID(data map[string]interface{}) string {
	roomID, _ := data["roomId"].(string)
	return roomID
}

func addToRoom(connection *Connection, roomID string) map[*websocket.Conn]string {
	mu.Lock()
	defer mu.Unlock()

	room, exists := rooms[roomID]
	if !exists {
		room = make(map[*websocket.Conn]string)
		rooms[roomID] = room
	}
	room[connection.conn] = connection.clientUUID

	return room
}

func notifyClient(connection *Connection, eventType, roomID string) {
	connection.send <- encodeJSON(map[string]interface{}{
		"type":   eventType,
		"roomId": roomID,
	})
}

func handleDevicesStatus(connection *Connection, data map[string]interface{}) {
	roomID := getRoomID(data)

	// `data["type"]` determina el estado del dispositivo
	updateDeviceStatus(connection, data["type"].(string))

	broadcastToRoom(roomID, map[string]interface{}{
		"type":     data["type"],
		"uuid":     connection.clientUUID,
		"cameraOn": connection.cameraOn,
		"audioOn":  connection.audioOn,
	}, connection.clientUUID)
}

func handleTranscript(data map[string]interface{}, connection *Connection) {
	roomID := getRoomID(data)
	messageType := data["type"].(string)
	messageText, ok := data["text"].(string)
	if !ok {
		messageText = ""
	}

	updateDeviceStatus(connection, messageType)

	message := map[string]interface{}{
		"type":     messageType,
		"uuid":     connection.clientUUID,
		"cameraOn": connection.cameraOn,
		"audioOn":  connection.audioOn,
		"message":  messageText,
	}

	broadcastToRoom(roomID, message, connection.clientUUID)
}

func updateDeviceStatus(connection *Connection, messageType string) {
	switch messageType {
	case "video_on_remote":
		connection.cameraOn = true
	case "video_off_remote":
		connection.cameraOn = false
	case "mic_on_remote":
		connection.audioOn = true
	case "mic_off_remote":
		connection.audioOn = false
	}
}

func getRoom(roomID string) map[*websocket.Conn]string {
	mu.Lock()
	defer mu.Unlock()

	return rooms[roomID]
}

func handleClientDisconnect(roomID string, connection *Connection) {
	removeFromRoom(roomID, connection)
	broadcastToRoom(roomID, map[string]interface{}{
		"type":   "close_call",
		"uuid":   connection.clientUUID,
		"roomId": roomID,
	}, connection.clientUUID)
}

func removeFromRoom(roomID string, connection *Connection) {
	mu.Lock()
	defer mu.Unlock()

	room := rooms[roomID]
	delete(room, connection.conn)
}

func removeRoom(roomID string) {
	mu.Lock()
	defer mu.Unlock()

	delete(rooms, roomID)
	fmt.Printf("Room %s removed as it became empty.\n", roomID)
}

func broadcastToRoom(roomID string, message map[string]interface{}, senderUUID string) {
	room := getRoom(roomID)

	for conn, clientUUID := range room {
		if clientUUID == senderUUID {
			continue
		}

		sendMessageToClient(conn, message, clientUUID)
	}

	if len(room) == 0 {
		removeRoom(roomID)
	}
}

func sendMessageToClient(conn *websocket.Conn, message map[string]interface{}, clientUUID string) {
	conn.SetWriteDeadline(time.Now().Add(writeWait))

	message["uuid"] = clientUUID

	if err := conn.WriteJSON(message); err != nil {
		log.Printf("Error sending message to client %s: %v", clientUUID, err)
		conn.Close()
	}
}

func encodeJSON(data map[string]interface{}) []byte {
	result, _ := json.Marshal(data)
	return result
}

func handleRoomMessage(data map[string]interface{}, connection *Connection) {
	roomID := getRoomID(data)

	message := map[string]interface{}{
		"type": "room_message",
		"uuid": connection.clientUUID,
		"text": data["text"].(string),
	}

	broadcastToRoom(roomID, message, connection.clientUUID)
}

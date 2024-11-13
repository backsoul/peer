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

// Variables globales
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1024
)

var (
	rooms = make(map[string]map[*websocket.Conn]string)
	mu    sync.Mutex
)

type Connection struct {
	conn       *websocket.Conn
	send       chan []byte
	clientUUID string
	cameraOn   bool
	audioOn    bool
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
	connection := initializeConnection(conn, clientUUID)

	// Configurar el tiempo de espera del pong
	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	go handleWrites(connection)

	sendUUIDToClient(connection, clientUUID)

	// Iniciar el temporizador de ping
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	// Manejar mensajes del cliente y enviar pings periódicos
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

func initializeConnection(conn *websocket.Conn, clientUUID string) *Connection {
	return &Connection{
		conn:       conn,
		send:       make(chan []byte, 512),
		clientUUID: clientUUID,
	}
}

func sendUUIDToClient(connection *Connection, clientUUID string) {
	connection.send <- encodeJSON(map[string]interface{}{
		"type": "uuid",
		"uuid": clientUUID,
	})
}

func handleClientMessages(conn *websocket.Conn, connection *Connection) {
	var roomID string
	for {
		messageType, data, err := readClientMessage(conn)
		if err != nil {
			handleClientError(connection, roomID, err)
			break
		}

		switch messageType {
		case "join":
			roomID = handleJoin(connection, data)
		case "close_call":
			handleClientDisconnect(roomID, connection)
		case "start_call", "webrtc_offer", "webrtc_answer", "webrtc_ice_candidate":
			handleRoomMessage(data, connection)
		case "transcript_text":
			handleTranscript(data, connection)
		case "mic_on_remote", "mic_off_remote", "video_on_remote", "video_off_remote":
			handleDevicesStatus(connection, data)
		default:
			log.Printf("Unknown message type: %s", messageType)
		}
	}
}

func readClientMessage(conn *websocket.Conn) (string, map[string]interface{}, error) {
	_, message, err := conn.ReadMessage()
	if err != nil {
		return "", nil, err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(message, &data); err != nil {
		return "", nil, fmt.Errorf("error unmarshaling message: %v", err)
	}

	messageType, ok := data["type"].(string)
	if !ok {
		return "", nil, fmt.Errorf("invalid message format")
	}

	return messageType, data, nil
}

func handleClientError(connection *Connection, roomID string, err error) {
	if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
		log.Printf("Client disconnected: %s", connection.clientUUID)
		handleClientDisconnect(roomID, connection)
	} else {
		log.Printf("Error reading message: %v", err)
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
	messageType := data["type"].(string)

	updateDeviceStatus(connection, messageType)

	// room := getRoom(roomID)
	broadcastToRoom(roomID, map[string]interface{}{
		"type":     messageType,
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
		messageText = "" // Asignar un valor por defecto si no está presente
	}

	// Actualizar el estado del dispositivo, aunque esto parece innecesario para transcripción
	updateDeviceStatus(connection, messageType)

	// Crear el mensaje que será retransmitido
	message := map[string]interface{}{
		"type":     messageType,
		"uuid":     connection.clientUUID,
		"cameraOn": connection.cameraOn,
		"audioOn":  connection.audioOn,
		"message":  messageText, // Aquí se incluye el texto de la transcripción
	}

	// Retransmitir el mensaje a todos los clientes en la sala, excepto al que lo envió
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
	if err := conn.WriteMessage(websocket.TextMessage, encodeJSON(message)); err != nil {
		log.Printf("Error broadcasting to client %s: %v", clientUUID, err)
		conn.Close()
		removeConnectionFromRoom(conn)
	} else {
		log.Printf("Message sent to client %s", clientUUID)
	}
}

func removeConnectionFromRoom(conn *websocket.Conn) {
	mu.Lock()
	defer mu.Unlock()

	for roomID, room := range rooms {
		delete(room, conn)
		if len(room) == 0 {
			delete(rooms, roomID)
		}
	}
}

func handleWrites(connection *Connection) {
	defer close(connection.send)

	for msg := range connection.send {
		if err := connection.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Printf("Error writing message: %v", err)
			break
		}
	}
}

func handleRoomMessage(data map[string]interface{}, connection *Connection) {
	roomID := getRoomID(data)
	room := getRoom(roomID)

	for client := range room {
		if client != connection.conn {
			data["from"] = connection.clientUUID
			data["audioOn"] = connection.audioOn
			data["cameraOn"] = connection.cameraOn
			client.WriteMessage(websocket.TextMessage, encodeJSON(data))
		}
	}
}

func encodeJSON(data map[string]interface{}) []byte {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error encoding JSON: %v", err)
		return nil
	}
	return jsonData
}

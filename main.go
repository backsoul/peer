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
	cameraOn   bool // Estado de la cámara
	audioOn    bool // Estado de la cámara
}

func main() {
	http.HandleFunc("/", handleConnection)

	port := "3000"
	fmt.Printf("Server listening on port %s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

// TODO: connection not is necesary send room,backend could calculate and asign.
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

	var roomID string // Variable para almacenar el roomID del cliente

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("Client disconnected: %s\n", clientUUID)
				if roomID != "" {
					// Enviar el mensaje de cierre de llamada a los otros clientes
					handleClientDisconnect(roomID, connection)
				}
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
		fmt.Println("messageType: %s data: ", messageType, data)
		switch messageType {
		case "join":
			roomID, _ = data["roomId"].(string) // Almacenar el roomID cuando se une
			handleJoin(connection, data)
		case "close_call":
			roomID, _ = data["roomId"].(string) // Almacenar el roomID cuando se une
			handleClientDisconnect(roomID, connection)
		case "start_call", "webrtc_offer", "webrtc_answer", "webrtc_ice_candidate":
			handleRoomMessage(data, connection)
		case "mic_on_remote", "mic_off_remote", "video_on_remote", "video_off_remote":
			handleDevicesStatus(connection, data)
		default:
			log.Printf("Unknown message type: %s\n", messageType)
		}
	}
}

func handleDevicesStatus(connection *Connection, data map[string]interface{}) {
	roomID, _ := data["roomId"].(string)
	messageType, _ := data["type"].(string)

	// Actualizar el estado de la cámara y micrófono según el mensaje recibido
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

	mu.Lock() // Bloquear el acceso concurrente a rooms
	room, exists := rooms[roomID]
	if !exists {
		room = make(map[*websocket.Conn]string)
		rooms[roomID] = room
	}
	room[connection.conn] = connection.clientUUID
	mu.Unlock() // Desbloquear

	// Log para verificar el estado antes de enviar
	fmt.Printf("Sending update to client %s: cameraOn=%t, audioOn=%t\n", connection.clientUUID, connection.cameraOn, connection.audioOn)

	// Difundir el nuevo estado a los otros clientes de la sala
	fmt.Println("Broadcasting updated state to other clients in the room")
	broadcast(roomID, map[string]interface{}{
		"type":     messageType,
		"uuid":     connection.clientUUID,
		"cameraOn": connection.cameraOn,
		"audioOn":  connection.audioOn,
	}, connection.clientUUID)
}

// Función para manejar la desconexión del cliente y notificar a los demás
func handleClientDisconnect(roomID string, connection *Connection) {
	mu.Lock()
	defer mu.Unlock()

	room, exists := rooms[roomID]
	if exists {
		// Eliminar al cliente del room
		delete(room, connection.conn)

		// Enviar un mensaje a los demás clientes de la sala notificando la desconexión
		if len(room) > 0 {
			broadcast(roomID, map[string]interface{}{
				"type":   "close_call",
				"uuid":   connection.clientUUID,
				"roomId": roomID,
			}, connection.clientUUID)
		} else {
			// Si no quedan más clientes, eliminar la sala
			delete(rooms, roomID)
		}
	}
}

func handleJoin(connection *Connection, data map[string]interface{}) {
	fmt.Println("entry join handle")
	roomID, _ := data["roomId"].(string)
	fmt.Println("entry join handle - roomID", roomID)
	mu.Lock() // Bloquear el acceso concurrente a rooms
	fmt.Println("entry join handle - after mu lock")
	room, exists := rooms[roomID]
	fmt.Println("room - len: ", len(room))
	fmt.Println("room - exists: ", exists)
	if !exists {
		room = make(map[*websocket.Conn]string)
		rooms[roomID] = room
	}

	room[connection.conn] = connection.clientUUID
	mu.Unlock() // Desbloquear
	fmt.Println("room - handleJoin: ", room)
	fmt.Println("room - data: ", data)

	// Notificar al cliente si la sala fue creada o si se unió
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

func broadcast(roomID string, message map[string]interface{}, senderUUID string) {
	mu.Lock() // Asegurarse de bloquear el acceso concurrente
	defer mu.Unlock()

	room, exists := rooms[roomID]
	if !exists {
		fmt.Println("Room does not exist")
		return
	}

	// Recorremos los clientes en el room
	for conn, clientUUID := range room {
		// Evitar enviar el mensaje al mismo cliente que lo envió
		if clientUUID == senderUUID {
			continue
		}

		// Enviar el mensaje a los demás clientes
		err := conn.WriteMessage(websocket.TextMessage, encodeJSON(message))
		if err != nil {
			fmt.Println("Error broadcasting message to client", clientUUID, ":", err)
			// Cerrar y eliminar la conexión con problemas
			conn.Close()
			delete(room, conn)
		} else {
			fmt.Println("Message sent to client", clientUUID)
		}
	}
}

func handleWrites(c *Connection) {
	defer func() {
		close(c.send) // Asegurarse de cerrar el canal de envío cuando la conexión se cierra
	}()

	for msg := range c.send {
		err := c.conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Println("Error writing message:", err)
			break
		}
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
				message["audioOn"] = connection.audioOn
				message["cameraOn"] = connection.cameraOn
				rooms[roomID][client] = connection.clientUUID
				client.WriteMessage(websocket.TextMessage, encodeJSON(message))
			}
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

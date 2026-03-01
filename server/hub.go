package server

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

//This is a user/seat.
// They are assigned a random id, have a connection and some messages.

type Client struct {
	id   string
	conn *websocket.Conn
	send chan []byte
}

// This is a room/bench.
// It exists when a seat is taken.
// Instead of schrondiger's, it could be izziebox (?) marketable.
type Room struct {
	id        string
	adminHash string
	clients   map[string]*Client //who's sitting on the bench
	mu        sync.Mutex         //lock before touch clients (ew wtf?)
}

// This is a hub. It has all rooms that have users in them. Kinda like a very noisy and loud park.
type Hub struct {
	rooms map[string]*Room
	mu    sync.Mutex // lock before touch rooms
}

// this functions is called once, when this whole thing starts, in main.go
func NewHub() *Hub {
	return &Hub{
		rooms: make(map[string]*Room),
	}
}

// this one checks if there's a room and if there isn't, it makes a new one.
// admin has is for new rooms only (HEHE), not for existing ones.
func (h *Hub) getOrCreateRoom(roomID, adminHash string) *Room {
	//locking the hub, because it reads or writes the room map
	h.mu.Lock()
	defer h.mu.Unlock()

	if room, exists := h.rooms[roomID]; exists {
		return room
	}

	room := &Room{
		id:        roomID,
		adminHash: adminHash,
		clients:   make(map[string]*Client),
	}
	h.rooms[roomID] = room
	return room
}

func (h *Hub) RoomSize(roomID string) int {
	h.mu.Lock()
	room, exists := h.rooms[roomID]
	h.mu.Unlock()

	if !exists {
		return 0
	}

	room.mu.Lock()
	defer room.mu.Unlock()
	return len(room.clients)
}

// this one adds a client to a room if it exists, otherwise creates the room and adds the client in
func (h *Hub) JoinRoom(roomID, adminHash string, client *Client) {
	room := h.getOrCreateRoom(roomID, adminHash) //Πέρνω μες το βράδυ, στο σκοτάδι, μα κανένα αγάπης σημάδι. 🎶

	//locking the room, this time, because it writes the client map
	room.mu.Lock()
	defer room.mu.Unlock()

	room.clients[client.id] = client
}

// this one makes client go bye-bye.
// if there's nobody on the bench, the bench gets pulled up by a secret tiny giant and eaten.
func (h *Hub) LeaveRoom(roomID string, client *Client) {
	h.mu.Lock()
	room, exists := h.rooms[roomID]
	h.mu.Unlock()

	if !exists {
		return
	}

	room.mu.Lock()
	delete(room.clients, client.id)
	isEmpty := len(room.clients) == 0
	room.mu.Unlock()

	if isEmpty {
		h.mu.Lock()
		delete(h.rooms, roomID)
		h.mu.Unlock()
	}
}

// this one is broadcasting a message.
// Sends it to everyone apart from the sender.
func (h *Hub) Broadcast(roomID string, senderID string, message []byte) {
	h.mu.Lock()
	room, exists := h.rooms[roomID]
	h.mu.Unlock()

	if !exists {
		return
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	for id, client := range room.clients {
		if id == senderID {
			continue //no going back to sender
		}

		// This part allows kind-of asynchronous reading of messages on "send" channel of each client.
		// And the rest of the clients don't need to wait for everyone to read the message broadcasted.
		select {
		case client.send <- message:
		default:
			close(client.send)
			delete(room.clients, id)
		}
	}
}

//this one verifies the admin token hash and closes/deletes the room.
//Since it's a bool out put, it's going to be true if the bench is gone or false if the admin token hash doesn't match.

// REVISION of this one later after completing the file.
func (h *Hub) DeleteRoom(roomID string, adminToken string) bool {
	h.mu.Lock()
	room, exists := h.rooms[roomID]
	h.mu.Unlock()

	if !exists {
		return false
	}

	//this is where it hashes the token and compares it to stored hash on the creation of the bench.
	if !verifyAdminToken(adminToken, room.adminHash) {
		return false
	}

	// send deleted message to everyone first, then wait for writePump to flush it
	// before closing channels — otherwise the close frame races the deleted message
	deletedMsg := buildOutgoing("deleted", "", "")
	room.mu.Lock()
	for _, client := range room.clients {
		select {
		case client.send <- deletedMsg:
		default:
		}
	}
	room.mu.Unlock()

	// give writePump time to flush the deleted message before closing
	time.Sleep(100 * time.Millisecond)

	// now close all channels — writePump catches this and disconnects
	room.mu.Lock()
	for _, client := range room.clients {
		close(client.send)
	}
	room.mu.Unlock()

	// tiny giant takes the bench away
	h.mu.Lock()
	delete(h.rooms, roomID)
	h.mu.Unlock()

	return true
}

// verifyAdminToken hashes the token incoming and checks it against the stored hash.
// If it doesn't match exactly how the client hashes it in crypto.js it's a noooooo goooo. :I
// And what it does is that it decodes the base64 into sha-256 bytes into hex string.
func verifyAdminToken(token, storedHash string) bool {
	tokenBytes, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		tokenBytes, err = base64.RawStdEncoding.DecodeString(token)
		if err != nil {
			log.Printf("DECODE ERROR: %v | token was: %s", err, token)
			return false
		}
	}
	hash := sha256.Sum256(tokenBytes)
	hexHash := hex.EncodeToString(hash[:])
	log.Printf("TOKEN: %s", token)
	log.Printf("COMPUTED HASH: %s", hexHash)
	log.Printf("STORED HASH:   %s", storedHash)
	return hexHash == storedHash
}

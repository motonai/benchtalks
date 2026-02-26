package server

import (
	"sync"
)

//This is a user/seat.
// They are assigned a random id, have a connection and some messages.

type Client struct {
	id   string
	conn *websocket.conn
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
		room: make(map[string]*Room),
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

// Broadcasting a message sends it to everyone apart from the sender
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
		//the meaning behind non-blocking send...
	}
}

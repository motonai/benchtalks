// So a big disclaimer here:
// This is the most goroutine-heavy thing in here.
// It's going to work as a translator between the Hubs and the Park.
// The hub speaks to local WS Clients and the Park uses NATS.
// The relay file will translate between them in both directions.

// So when a message crosses from one bench to another through NATS, it can't be
// the raw encrypted blob.
// The relay wraps it, and then decides (via the user) on which bench to send
// it, in order to prevent loops.

// The main goroutine will be establishing the NATS connection and then it will
// register the subscription handler.
// The NATS library has it's one internal goroutines, calling the bench's
// handler function whenever a park public message arrives & the bench's handler
// checks benchID, calls the hub's Broadcast from Park.
// The last but not least goroutine is from the bench's hub: calling the
// relay.Publish() when broadcasting a public room message.

// Also the relay itself doesn't do stuff directly. NATS manages it's own.

package nats

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/nats-io/nats.go"
)

// This is the NATS subject namespace for all public room messages
const subjectPrefix = "room.public."

// This one is the envelope I told you in the disclaimer
type ParkMessage struct {
	BenchID string `json:"benchId"` //which bench sent this for the first time
	Payload []byte `json:"payload"` // untouched encrypted blob
}

// This one is what the relay calls when a message arrives that should be
// forwarded to local clients.
// It matches the hub's Broadcast from Park. It's defined as a function type
// here so relay doesn't need to import the hub package.
// I don't want a circular import created.
type BroadcastFunc func(roomID string, payload []byte)

// The relay struct manages the NATS cluster connection, and also handles
// publishing and receiving messages for public rooms.
type Relay struct {
	conn      *nats.Conn    // live connection to NATS cluster
	benchID   string        // buid
	broadcast BroadcastFunc // called when a park message arrives for local clients
}

// This one creates the connection to NATS cluster using the peer addresses,
// then registers the subscription handler for all public rooms.
// Gives back a plug&play relay or an error if the connection fails.

// Reconnection is managed automatically.
func Connect(peers []string, benchID string, broadcast BroadcastFunc) (*Relay, error) {
	// nats.go wants a single URL string when connecting to a cluster (that
	// means multiple addresses joined with commas)
	url := strings.Join(peers, ",")

	//NATS handsake and tcp connection are handled by "nats.Connect"
	//The options make it keep retrying indefinitely, if a peer is down.
	conn, err := nats.Connect(url,
		nats.MaxReconnects(-1),

		//logging losing connections and reconnecting - debuging reasons
		//NO MESSAGE CONTENT - only peering issues
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			log.Printf("[nats] disconnected from park: %v", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("[nats] reconnected to park via %s", nc.ConnectedUrl())
		}),
	)
	if err != nil {
		return nil, err
	}

	r := &Relay{
		conn:      conn,
		benchID:   benchID,
		broadcast: broadcast,
	}

	// Registration of subscriptions. NATS will call r.handleIncoming whenever a
	// message arrives on room.public.* (any public room and bench)
	// * stands for wildcard, and if it matches exactly one path, as examples:
	// "room.public.abc" and "room.public.xyz" both match, but
	// "room.public.abc.extra" doesn't match.
	if _, err := conn.Subscribe(subjectPrefix+"*", r.handleIncoming); err != nil {
		conn.Close()
		return nil, err
	}

	log.Printf("[nats] connected to park, bench ID: %s", benchID)
	return r, nil
}

// This one sends a message for a public room into the park.
// Called by hub's Broadcast when a room is public and a relay is configured.
// relay doesn't read or modify payload! (important)
func (r *Relay) Publish(roomID string, payload []byte) {
	msg := ParkMessage{
		BenchID: r.benchID,
		Payload: payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		// This is a 100% unhappening thing. ParkMessage only contains a string
		// and bytes
		log.Printf("[nats] failed to marshal park message: %v", err)
		return
	}

	subject := subjectPrefix + roomID
	if err := r.conn.Publish(subject, data); err != nil {
		log.Printf("[nats] failed to publish to %s: %v", subject, err)
	}
}

// This one is called by NATS library (in its own goroutine), when a message
// gets to room.public.*.
// Checks the BenchID to prevent loops, and then it forwards the payload to
// local clients via broadcasting
func (r *Relay) handleIncoming(msg *nats.Msg) {
	var park ParkMessage
	if err := json.Unmarshal(msg.Data, &park); err != nil {
		log.Printf("[nats] received malformed park message on %s: %v", msg.Subject, err)
		return
	}

	// loop prevention. the message being send from this bench to the park via
	// NATS, will come back to us too (since we're subscribed to our own
	// subject)
	// that's why we discard it here, so local clients don't see duplicates of
	// their own messages.
	if park.BenchID == r.benchID {
		return
	}

	// Pull the room ID from subject and trim prefix removes the "room.public"
	// leaving the message
	roomID := strings.TrimPrefix(msg.Subject, subjectPrefix)
	if roomID == "" {
		return
	}

	// Get it to clients while hub finds the right room, in each connected WS
	// client
	r.broadcast(roomID, park.Payload)
}

// This one closes the NATS connection and is called when the server shuts down.
func (r *Relay) Close() {
	if r.conn != nil {
		r.conn.Close()
		log.Printf("[nats] disconnected from park")
	}
}

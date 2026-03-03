package server

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"time"
)

const Version = "1.2.0"

var startTime = time.Now()

// Health Handler serves "GET /health" - a public JSON ss of server state.
// The CORS header lets status.benchtalks.chat get this from a different origin.
func (h *Hub) HealthHandler(w http.ResponseWriter, r *http.Request) {
	rooms, users := h.HealthSnapshot()

	//asking relay if it's coonected, nil means it's standalone
	natsConnected := h.relay != nil && h.relay.IsConnected()

	//response
	payload := map[string]any{
		"status":         "ok",
		"version":        Version,
		"bench_id":       h.cfg.BenchID,
		"uptime_seconds": int(time.Since(startTime).Seconds()),
		"active_rooms":   rooms,
		"total_users":    users,
		"nats_connected": natsConnected,
		"nats_peers":     h.cfg.NATSPeers,
		"websocket_ok":   true,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(payload)
}

// So Go embeds the entire public/ folder into the binary.
func NewRouter(hub *Hub, staticFiles embed.FS) http.Handler {
	mux := http.NewServeMux()

	// take off "public" as prefix so /public/index.html is served as /index.html
	// fs.Sub makes the "prefix" go away and it makes things get served from root, like the JS version.
	stripped, err := fs.Sub(staticFiles, "public")
	if err != nil {
		// if the filesystem embedded in, is broken, the server is going to have a tummy ache.
		// So instead of pushing it to try again or start up and serve nothing,
		// it's better to make it crash immediately
		panic("could not strip public/ prefix from embedded files: " + err.Error())
	}

	fileServer := http.FileServer(http.FS(stripped))

	// ws endpoint c:
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ServeWS(hub, w, r)
	})

	//health :D
	//we need to do it here before the "/" because after it, the catchall would intercept the request and serve something that doesn't exist, making it a 404.
	mux.HandleFunc("/health", hub.HealthHandler)
	// Read index.html directly from the embedded FS and serve it ourselves.
	// We bypass the file server entirely for this one file because Go's file server
	// always redirects /index.html → / which causes an infinite loop.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "" {
			data, err := fs.ReadFile(stripped, "index.html")
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(data)
			return
		}
		fileServer.ServeHTTP(w, r)
	})

	return mux
}

// since the go:embed directive needs to be in the same package as the variable that holds the files,
// the //go:embed line lives in main.go instead of here.

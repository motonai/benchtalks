package server

import (
	"embed"
	"io/fs"
	"net/http"
)

// So Go embeds the entire public/ folder into the binary.
func NewRouter(hub *Hub, staticFiles embed.FS) http.Handler {
	mux := http.NewServeMux()

	// take off "public" as prefix so /public/index.html is served as
	// /index.html
	// fs.Sub makes the "prefix" go away and it makes things get served from
	// root, like the JS version.
	stripped, err := fs.Sub(staticFiles, ".")
	if err != nil {
		// If the filesystem embedded in, is broken, the server is going to have
		// a tummy ache.
		// So instead of pushing it to try again or start up and serve nothing,
		// it's better to make it crash immediately
		panic("could not strip public/ prefix from embedded files: " + err.Error())
	}

	fileServer := http.FileServer(http.FS(stripped))

	// ws endpoint c:
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ServeWS(hub, w, r)
	})

	// Read index.html directly from the embedded FS and serve it ourselves.
	// We bypass the file server entirely for this one file because Go's file
	// server
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

// Since the go:embed directive needs to be in the same package as the variable
// that holds the files,
// the //go:embed line lives in main.go instead of here.
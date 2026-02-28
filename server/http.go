package server

import (
	"embed"
	"io/fs"
	"net/http"
)

// So Go embeds the entire public/ folder into the binary.
func NewRouter(hub *Hub, staticFiles embed.FS) http.Handler {
	mux := http.NewServeMux()

	//take off "public" as prefix removal and the /server/index.html is served as /index.html
	//fs.Sub makes the "prefix" go away and it makes things get served from root, like the JS version.
	stripped, err := fs.Sub(staticFiles, "public")
	if err != nil {
		//if the filesystem embedded in, is broken, the server is going to have a tummy ache. So instead of pushing it to try again or start up and serve nothing, it's better to make it crash immidiatelly
		panic("could not strip public/ prefix  from embedded files: " + err.Error())
	}

	//serve static files at root
	mux.Handle("/", http.FileServer(http.FS(stripped)))

	//ws endpoint c:
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ServeWS(hub, w, r)
	})

	return mux
}

//since the go:embed directive needs to be in the same package as the variable that holds the files.

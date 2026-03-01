package main

import (
	"embed"
	"log"
	"net/http"

	"github.com/isidman/benchtalks/config"
	"github.com/isidman/benchtalks/server"
)

//go:embed all:public
var staticFiles embed.FS

func main() {
	cfg := config.Load()
	hub := server.NewHub()

	router := server.NewRouter(hub, staticFiles)

	log.Printf("Benchtalks is listening on: %s", cfg.Port)

	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatal("server error:", err)
	}
}

package main

import (
	"log"
	"net/http"
)

func main() {
	srv, err := NewServer("posts", "templates")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", srv.HandleHealth)
	mux.HandleFunc("/", srv.HandleIndex)
	mux.HandleFunc("/posts/", srv.HandlePost)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

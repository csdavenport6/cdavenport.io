package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
)

//go:embed static
var staticFiles embed.FS

//go:embed Connor-Davenport-Engineering.pdf
var resumePDF []byte

func main() {
	srv, err := NewServer("posts", "templates")
	if err != nil {
		log.Fatal(err)
	}

	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", srv.HandleHealth)
	mux.HandleFunc("/resume", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", `inline; filename="Connor-Davenport-Engineering.pdf"`)
		_, _ = w.Write(resumePDF)
	})
	mux.HandleFunc("/", srv.HandleIndex)
	mux.HandleFunc("/posts/", srv.HandlePost)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

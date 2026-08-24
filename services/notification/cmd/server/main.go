package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("NOTIFICATION_PORT")
	if port == "" {
		port = "8085"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("notification ok"))
	})

	log.Printf("notification listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

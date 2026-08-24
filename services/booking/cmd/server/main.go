package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("BOOKING_PORT")
	if port == "" {
		port = "8083"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("booking ok"))
	})

	log.Printf("booking listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

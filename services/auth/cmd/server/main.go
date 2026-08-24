package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("AUTH_PORT")
	if port == "" {
		port = "8081"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("auth ok"))
	})

	log.Printf("auth listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

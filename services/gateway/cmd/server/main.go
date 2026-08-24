package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("GATEWAY_PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("gateway ok"))
	})

	log.Printf("gateway listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("CATALOG_PORT")
	if port == "" {
		port = "8082"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("catalog ok"))
	})

	log.Printf("catalog listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

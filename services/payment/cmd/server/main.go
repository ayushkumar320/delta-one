package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PAYMENT_PORT")
	if port == "" {
		port = "8084"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("payment ok"))
	})

	log.Printf("payment listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

//go:build legacy
// +build legacy

package main

import (
	"log"
	"math/rand"
	"net/http"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	initCategoryConfig()

	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	log.Println("Mahjong WebSocket server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

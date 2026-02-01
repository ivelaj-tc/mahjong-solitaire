package main

import (
	"log"
	"net/http"

	"mahjong-backend/internal/config"
	"mahjong-backend/internal/ws"
)

func main() {
	// rand.Seed(time.Now().UnixNano())
	categorySymbols, categoryFileTypes, availableCategories := config.InitCategoryConfig()

	server := ws.NewServer(categorySymbols, categoryFileTypes, availableCategories)

	http.HandleFunc("/ws", server.Handler)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	log.Println("Mahjong WebSocket server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

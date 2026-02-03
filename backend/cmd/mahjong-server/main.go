package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"mahjong-backend/internal/config"
	"mahjong-backend/internal/store"
	"mahjong-backend/internal/ws"
)

func main() {
	// rand.Seed(time.Now().UnixNano())
	categorySymbols, categoryFileTypes, availableCategories := config.InitCategoryConfig()
	redisStore := initRedisStore()

	server := ws.NewServer(categorySymbols, categoryFileTypes, availableCategories, redisStore)

	http.HandleFunc("/ws", server.Handler)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	log.Println("Mahjong WebSocket server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func initRedisStore() *store.RedisStore {
	db := 0
	if dbStr := os.Getenv("REDIS_DB"); dbStr != "" {
		if parsed, err := strconv.Atoi(dbStr); err == nil {
			db = parsed
		} else {
			log.Printf("Invalid REDIS_DB %q: %v", dbStr, err)
		}
	}

	sentinelAddrs := splitCSVEnv("REDIS_SENTINEL_ADDRS")
	if len(sentinelAddrs) > 0 {
		masterName := os.Getenv("REDIS_SENTINEL_MASTER")
		if masterName == "" {
			masterName = "mahjong-master"
		}
		options := &redis.FailoverOptions{
			MasterName:       masterName,
			SentinelAddrs:    sentinelAddrs,
			Password:         os.Getenv("REDIS_PASSWORD"),
			SentinelPassword: os.Getenv("REDIS_SENTINEL_PASSWORD"),
			DB:               db,
		}
		client := redis.NewFailoverClient(options)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := client.Ping(ctx).Err(); err != nil {
			_ = client.Close()
			log.Printf("Redis Sentinel disabled (ping failed): %v", err)
			return nil
		}
		log.Printf("Redis Sentinel enabled (master %s) at %s", masterName, strings.Join(sentinelAddrs, ","))
		return store.NewRedisStore(client, resolveRedisTTL())
	}

	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		return nil
	}

	options := &redis.Options{
		Addr:     addr,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       db,
	}
	client := redis.NewClient(options)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		log.Printf("Redis disabled (ping failed): %v", err)
		return nil
	}

	log.Printf("Redis enabled at %s", addr)
	return store.NewRedisStore(client, resolveRedisTTL())
}

func resolveRedisTTL() time.Duration {
	ttl := store.DefaultRoomStateTTL
	if ttlValue := os.Getenv("REDIS_TTL"); ttlValue != "" {
		if parsed, err := time.ParseDuration(ttlValue); err == nil {
			return parsed
		}
		log.Printf("Invalid REDIS_TTL %q", ttlValue)
	}
	if ttlHours := os.Getenv("REDIS_TTL_HOURS"); ttlHours != "" {
		if hours, err := strconv.Atoi(ttlHours); err == nil {
			return time.Duration(hours) * time.Hour
		}
		log.Printf("Invalid REDIS_TTL_HOURS %q", ttlHours)
	}
	return ttl
}

func splitCSVEnv(name string) []string {
	value := os.Getenv(name)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	trimmed := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			trimmed = append(trimmed, part)
		}
	}
	return trimmed
}

package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	DefaultRedisKeyPrefix = "mahjong:room:"
	DefaultRedisChannel   = "mahjong:room-events"
	DefaultWaitingRoomKey = "mahjong:waiting-room"
	DefaultRoomStateTTL   = 24 * time.Hour
)

type RedisStore struct {
	client    *redis.Client
	ttl       time.Duration
	keyPrefix string
	channel   string
}

func (s *RedisStore) GetWaitingRoomID(ctx context.Context) (string, bool, error) {
	value, err := s.client.Get(ctx, s.waitingRoomKey()).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return value, true, nil
}

func (s *RedisStore) SetWaitingRoomID(ctx context.Context, roomID string) (bool, error) {
	return s.client.SetNX(ctx, s.waitingRoomKey(), roomID, s.ttl).Result()
}

func (s *RedisStore) ClearWaitingRoomID(ctx context.Context, roomID string) error {
	if roomID == "" {
		return nil
	}
	value, err := s.client.Get(ctx, s.waitingRoomKey()).Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return err
	}
	if value != roomID {
		return nil
	}
	return s.client.Del(ctx, s.waitingRoomKey()).Err()
}

func NewRedisStore(client *redis.Client, ttl time.Duration) *RedisStore {
	if ttl <= 0 {
		ttl = DefaultRoomStateTTL
	}
	return &RedisStore{
		client:    client,
		ttl:       ttl,
		keyPrefix: DefaultRedisKeyPrefix,
		channel:   DefaultRedisChannel,
	}
}

func (s *RedisStore) SaveRoom(ctx context.Context, roomID string, snapshot GameSnapshot) error {
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, s.roomKey(roomID), payload, s.ttl).Err()
}

func (s *RedisStore) LoadRoom(ctx context.Context, roomID string) (GameSnapshot, bool, error) {
	value, err := s.client.Get(ctx, s.roomKey(roomID)).Result()
	if err == redis.Nil {
		return GameSnapshot{}, false, nil
	}
	if err != nil {
		return GameSnapshot{}, false, err
	}
	var snapshot GameSnapshot
	if err := json.Unmarshal([]byte(value), &snapshot); err != nil {
		return GameSnapshot{}, false, err
	}
	return snapshot, true, nil
}

func (s *RedisStore) PublishRoomState(ctx context.Context, state RoomState) error {
	payload, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return s.client.Publish(ctx, s.channel, payload).Err()
}

func (s *RedisStore) SubscribeRoomUpdates(ctx context.Context, handler func(RoomState)) error {
	pubsub := s.client.Subscribe(ctx, s.channel)
	defer pubsub.Close()

	if _, err := pubsub.Receive(ctx); err != nil {
		return err
	}

	receiveChannel := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case message, ok := <-receiveChannel:
			if !ok {
				return nil
			}
			var state RoomState
			if err := json.Unmarshal([]byte(message.Payload), &state); err != nil {
				continue
			}
			handler(state)
		}
	}
}

func (s *RedisStore) roomKey(roomID string) string {
	return s.keyPrefix + roomID
}

func (s *RedisStore) waitingRoomKey() string {
	return DefaultWaitingRoomKey
}

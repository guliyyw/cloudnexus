package captcha

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore implements base64Captcha.Store using Redis.
type RedisStore struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
}

func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{
		client: client,
		prefix: "captcha:",
		ttl:    5 * time.Minute,
	}
}

func (s *RedisStore) Set(id string, value string) {
	s.client.Set(context.Background(), s.prefix+id, value, s.ttl)
}

func (s *RedisStore) Get(id string, clear bool) string {
	key := s.prefix + id
	val, err := s.client.Get(context.Background(), key).Result()
	if err != nil {
		return ""
	}
	if clear {
		s.client.Del(context.Background(), key)
	}
	return val
}

func (s *RedisStore) Verify(id, answer string, clear bool) bool {
	v := s.Get(id, clear)
	return v != "" && v == answer
}

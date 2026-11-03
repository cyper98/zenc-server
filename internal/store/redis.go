package store

import (
	"context"
	"fmt"
	"time"

	"github.com/cyper98/zenc-server/pkg/config"
	"github.com/redis/go-redis/v9"
)

// RedisStore provides Redis-based storage operations
type RedisStore struct {
	client redis.UniversalClient
	ctx    context.Context
}

// NewRedisStore creates a new Redis store instance
func NewRedisStore(cfg config.RedisConfig) (*RedisStore, error) {
	var client redis.UniversalClient

	switch cfg.Mode {
	case "single":
		client = redis.NewClient(&redis.Options{
			Addr:     cfg.Addresses[0],
			Password: cfg.Password,
			DB:       cfg.DB,
		})
	case "cluster":
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    cfg.Addresses,
			Password: cfg.Password,
		})
	case "sentinel":
		// For sentinel mode, first address is the master name
		masterName := cfg.Addresses[0]
		sentinelAddrs := cfg.Addresses[1:]
		client = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    masterName,
			SentinelAddrs: sentinelAddrs,
			Password:      cfg.Password,
			DB:            cfg.DB,
		})
	default:
		return nil, fmt.Errorf("unsupported Redis mode: %s", cfg.Mode)
	}

	ctx := context.Background()

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisStore{
		client: client,
		ctx:    ctx,
	}, nil
}

// Close closes the Redis connection
func (s *RedisStore) Close() error {
	return s.client.Close()
}

// Set stores a key-value pair with optional expiration
func (s *RedisStore) Set(key string, value interface{}, expiration time.Duration) error {
	return s.client.Set(s.ctx, key, value, expiration).Err()
}

// Get retrieves a value by key
func (s *RedisStore) Get(key string) (string, error) {
	return s.client.Get(s.ctx, key).Result()
}

// Delete removes a key
func (s *RedisStore) Delete(key string) error {
	return s.client.Del(s.ctx, key).Err()
}

// Exists checks if a key exists
func (s *RedisStore) Exists(key string) (bool, error) {
	count, err := s.client.Exists(s.ctx, key).Result()
	return count > 0, err
}

// HSet sets a hash field
func (s *RedisStore) HSet(key string, field string, value interface{}) error {
	return s.client.HSet(s.ctx, key, field, value).Err()
}

// HGet gets a hash field value
func (s *RedisStore) HGet(key string, field string) (string, error) {
	return s.client.HGet(s.ctx, key, field).Result()
}

// HGetAll gets all hash fields
func (s *RedisStore) HGetAll(key string) (map[string]string, error) {
	return s.client.HGetAll(s.ctx, key).Result()
}

// HDel deletes hash fields
func (s *RedisStore) HDel(key string, fields ...string) error {
	return s.client.HDel(s.ctx, key, fields...).Err()
}

// Publish publishes a message to a channel
func (s *RedisStore) Publish(channel string, message interface{}) error {
	return s.client.Publish(s.ctx, channel, message).Err()
}

// Subscribe subscribes to channels
func (s *RedisStore) Subscribe(channels ...string) *redis.PubSub {
	return s.client.Subscribe(s.ctx, channels...)
}

// SAdd adds members to a set
func (s *RedisStore) SAdd(key string, members ...interface{}) error {
	return s.client.SAdd(s.ctx, key, members...).Err()
}

// SRem removes members from a set
func (s *RedisStore) SRem(key string, members ...interface{}) error {
	return s.client.SRem(s.ctx, key, members...).Err()
}

// SMembers gets all members of a set
func (s *RedisStore) SMembers(key string) ([]string, error) {
	return s.client.SMembers(s.ctx, key).Result()
}

// SIsMember checks if a member exists in a set
func (s *RedisStore) SIsMember(key string, member interface{}) (bool, error) {
	return s.client.SIsMember(s.ctx, key, member).Result()
}

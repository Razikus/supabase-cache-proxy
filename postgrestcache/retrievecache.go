package postgrestcache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"net/url"
	"sort"
	"strings"
	"time"
)

type RedisCacherro struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisCacherro(client *redis.Client, ttl time.Duration) *RedisCacherro {
	return &RedisCacherro{
		client: client,
		ttl:    ttl,
	}
}

func (r *RedisCacherro) generateCacheKey(path string, headers map[string][]string, query url.Values) string {
	components := []string{path}

	if len(query) > 0 {
		queryKeys := make([]string, 0, len(query))
		for k := range query {
			queryKeys = append(queryKeys, k)
		}
		sort.Strings(queryKeys)

		for _, k := range queryKeys {
			values := query[k]
			sort.Strings(values)
			components = append(components, fmt.Sprintf("%s=%s", k, strings.Join(values, ",")))
		}
	}

	relevantHeaders := []string{"Authorization", "Accept", "Content-Type", "Apikey"}
	for _, header := range relevantHeaders {
		if vals, ok := headers[header]; ok {
			components = append(components, fmt.Sprintf("%s=%s", header, strings.Join(vals, ",")))
		}
	}

	key := strings.Join(components, "|")

	hash := sha256.Sum256([]byte(key))
	return "postgrest:" + hex.EncodeToString(hash[:])
}

func (r *RedisCacherro) Get(ctx context.Context, path string, headers map[string][]string, query url.Values) (*CachedResponse, int, error) {
	key := r.generateCacheKey(path, headers, query)

	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, -1, nil
		}
		return nil, -1, fmt.Errorf("redis get error: %w", err)
	}

	var cached CachedResponse
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, -1, fmt.Errorf("unmarshal error: %w", err)
	}

	return &cached, len(data), nil
}

func (r *RedisCacherro) Set(ctx context.Context, path string, headers map[string][]string, query url.Values, response *CachedResponse) error {
	key := r.generateCacheKey(path, headers, query)

	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	if err := r.client.Set(ctx, key, data, r.ttl).Err(); err != nil {
		return fmt.Errorf("redis set error: %w", err)
	}

	return nil
}

func (r *RedisCacherro) Delete(ctx context.Context, path string, headers map[string][]string, query url.Values) error {
	key := r.generateCacheKey(path, headers, query)

	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis delete error: %w", err)
	}

	return nil
}

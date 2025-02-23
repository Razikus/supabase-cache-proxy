package postgrestcache

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

type Cacher struct {
	SupaURL       string
	client        *http.Client
	ReverseProxy  *httputil.ReverseProxy
	RedisClient   *redis.Client
	RedisCacherro *RedisCacherro
	TablesToCache []string
}

func NewCacher(supaURL string, rdb *redis.Client, toCache []string, ttl time.Duration) *Cacher {
	toUrl, err := url.Parse(supaURL)
	if err != nil {
		panic(err)
	}
	cacherro := NewRedisCacherro(rdb, ttl)
	proxier := httputil.NewSingleHostReverseProxy(toUrl)
	originalDirector := proxier.Director
	proxier.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = req.URL.Host
	}

	return &Cacher{
		SupaURL:       supaURL,
		client:        &http.Client{},
		ReverseProxy:  proxier,
		RedisClient:   rdb,
		RedisCacherro: cacherro,
		TablesToCache: toCache,
	}
}

func (c *Cacher) shouldCache(what string) bool {
	lastPartOf := strings.Split(what, "/")
	lastElement := lastPartOf[len(lastPartOf)-1]
	for _, table := range c.TablesToCache {
		if table == "*" {
			return true
		}
		if strings.ToUpper(table) == strings.ToUpper(lastElement) {
			return true
		}

	}
	return false
}

func (c *Cacher) RegisterHandler() *http.ServeMux {
	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Info().Str("Req", r.URL.Path).Str("Method", r.Method).Str("UserAgent", r.UserAgent()).Send()
		if len(c.TablesToCache) > 0 {
			if strings.HasPrefix(r.URL.Path, "/rest/v1/") {
				if r.Method == http.MethodGet {
					if c.shouldCache(r.URL.Path) {
						ctx := context.Background()

						what, savedbytes, err := c.RedisCacherro.Get(ctx, r.URL.Path, r.Header, r.URL.Query())
						if what != nil && err == nil {
							log.Info().Int("Saved", savedbytes).Str("CacheHit", r.URL.Path).Send()
							what.WriteTo(w)
							return
						}

						cachedWriter := NewCacheResponseWriter(w)
						c.ReverseProxy.ServeHTTP(cachedWriter, r)

						serialized := cachedWriter.ToCachedResponse()
						err = c.RedisCacherro.Set(ctx, r.URL.Path, r.Header, r.URL.Query(), serialized)
						if err == nil {
							log.Info().Str("CacheSave", r.URL.Path).Send()
						} else {
							log.Error().Err(err).Str("CacheSave", r.URL.Path).Send()
						}

						return
					} else {
						c.ReverseProxy.ServeHTTP(w, r)
						return
					}
				}

			}
		}
		c.ReverseProxy.ServeHTTP(w, r)
		return
	})
	return handler
}

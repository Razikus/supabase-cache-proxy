package main

import (
	"github.com/Razikus/postgrest-cache-redis/postgrestcache"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseTables(tablesEnv string) []string {
	if tablesEnv == "" {
		return []string{"*"} // Default to cache all tables
	}
	return strings.Split(tablesEnv, ",")
}

func parseTTL(ttlMinutes string) time.Duration {
	minutes, err := strconv.Atoi(ttlMinutes)
	if err != nil || minutes <= 0 {
		return time.Minute * 5 // Default 5 minutes
	}
	return time.Minute * time.Duration(minutes)
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	cacheUrl := getEnvOrDefault("SUPA_URL", "http://localhost:3000")
	port := getEnvOrDefault("PORT", "8080")

	// Redis configuration
	redisAddr := getEnvOrDefault("REDIS_ADDR", "localhost:6379")
	redisPassword := getEnvOrDefault("REDIS_PASSWORD", "")
	redisDB, _ := strconv.Atoi(getEnvOrDefault("REDIS_DB", "0"))

	// Cache configuration
	cacheTTL := parseTTL(getEnvOrDefault("CACHE_TTL_MINUTES", "5"))
	toCache := parseTables(getEnvOrDefault("CACHE_TABLES", "*"))

	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})

	proxy := postgrestcache.NewCacher(cacheUrl, redisClient, toCache, cacheTTL)
	registered := proxy.RegisterHandler()

	log.Info().Str("LIFECYCLE", "STARTING REVERSE CACHER").Send()
	log.Info().Str("URL", cacheUrl).Send()
	log.Info().Str("REDIS", redisAddr).Send()
	log.Info().Dur("TTL", cacheTTL).Send()
	log.Info().Strs("TABLES", toCache).Send()
	log.Info().Str("PORT", port).Send()

	log.Info().Msgf("Starting server on :%s", port)
	if err := http.ListenAndServe(":"+port, registered); err != nil {
		log.Error().Err(err).Msg("server failed to start")
		os.Exit(1)
	}
}

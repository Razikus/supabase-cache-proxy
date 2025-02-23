package main

import (
	"fmt"
	"github.com/Razikus/postgrest-cache-redis/postgrestcache"
	"github.com/redis/go-redis/v9"
	"log"
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

	fmt.Println("STARTING REVERSE CACHER:")
	fmt.Println("URL:", cacheUrl)
	fmt.Println("REDIS:", redisAddr)
	fmt.Println("TTL:", cacheTTL)
	fmt.Println("TABLES:", toCache)
	fmt.Println("PORT:", port)

	log.Printf("Starting server on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, registered))

}

package main

import (
	"api-key-limiter/handlers"
	"api-key-limiter/middleware"
	"api-key-limiter/proxy"
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-redis/redis_rate/v10"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

func connectToDb() *sql.DB {
	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	user := os.Getenv("POSTGRES_USER")
	pass := os.Getenv("POSTGRES_PASS")
	dbName := os.Getenv("POSTGRES_DB")

	conn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, pass, dbName,
	)

	db, err := sql.Open("postgres", conn)
	if err != nil {
		log.Fatalf("failed to connect to db: %v\n", err)
	}

	return db
}

func connectToRedis() *redis.Client {
	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")
	return redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", host, port),
	})
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	db := connectToDb()
	rdb := connectToRedis()

	if err := rdb.FlushDB(context.Background()).Err(); err != nil {
		log.Fatalf("failed to flush redis: %v\n", err)
	}

	limiter := redis_rate.NewLimiter(rdb)
	projectHandler := handlers.NewProjectHandler(db)
	authMiddleware := middleware.NewAuthMiddleware(projectHandler)

	url := "localhost:9000"
	proxy, proxyErr := proxy.NewProxy(projectHandler, limiter)

	if proxyErr != nil {
		log.Fatalf("failed to create proxy: %v\n", proxyErr)
	}

	log.Printf("Starting Proxy server on %s\n", url)
	log.Fatal(http.ListenAndServeTLS(url, "certs/ca.pem", "certs/ca.key.pem", authMiddleware.Auth(proxy)))
}

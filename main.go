package main

import (
	"api-key-limiter/handlers"
	"api-key-limiter/middleware"
	"api-key-limiter/proxy"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func connectToDb() *sql.DB {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

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
		panic(err)
	}

	return db
}

func main() {
	db := connectToDb()
	projectHandler := handlers.NewProjectHandler(db)
	authMiddleware := middleware.NewAuthMiddleware(projectHandler)

	url := "0.0.0.0:9000"
	proxy, proxyErr := proxy.NewProxy(projectHandler)

	if proxyErr != nil {
		log.Fatalf("failed to create proxy: %v\n", proxyErr)
	}

	log.Printf("Starting Proxy server on %s\n", url)
	log.Fatal(http.ListenAndServeTLS(url, "certs/ca.pem", "certs/ca.key.pem", authMiddleware.Auth(proxy)))
}

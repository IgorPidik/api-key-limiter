package main

import (
	"api-key-limiter/proxy"
	"log"
	"net/http"
)

func main() {
	url := "0.0.0.0:8000"
	log.Printf("Starting Proxy server on %s\n", url)
	proxy, proxyErr := proxy.NewProxy()

	if proxyErr != nil {
		log.Fatalf("failed to create proxy: %w\n", proxyErr)
	}

	log.Fatal(http.ListenAndServe(url, proxy))
}

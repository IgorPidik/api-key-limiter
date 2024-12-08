package main

import (
	"api-key-limiter/proxy"
	"log"
	"net/http"
)

func main() {
	url := "0.0.0.0:9000"
	log.Printf("Starting Proxy server on %s\n", url)
	proxy, proxyErr := proxy.NewProxy()

	if proxyErr != nil {
		log.Fatalf("failed to create proxy: %w\n", proxyErr)
	}

	log.Fatal(http.ListenAndServeTLS(url, "certs/ca.pem", "certs/ca.key.pem", proxy))
}

package main

import (
	"api-key-limiter/proxy"
	"log"
	"net/http"
)

func main() {
	url := "0.0.0.0:8000"
	log.Printf("Starting Proxy server on %s\n", url)
	proxy := proxy.NewProxy()
	log.Fatal(http.ListenAndServe(url, proxy))
}

package main

import (
	"api-key-limiter/proxy"
	"fmt"
	"log"
	"net/http"
)

func main() {
	url := "localhost:8000"
	fmt.Printf("Starting Proxy server on %s\n", url)
	proxy := proxy.NewProxy()
	log.Fatal(http.ListenAndServe(url, proxy))
}

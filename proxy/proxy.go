package proxy

import (
	"api-key-limiter/handlers"
	"api-key-limiter/models"
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"time"

	"github.com/go-redis/redis_rate/v10"
)

type Proxy struct {
	cert           *tls.Certificate
	projectHandler *handlers.ProjectHandler
	limiter        *redis_rate.Limiter
}

func NewProxy(projectHandler *handlers.ProjectHandler, limiter *redis_rate.Limiter) (*Proxy, error) {
	cert, certErr := loadCA()
	if certErr != nil {
		return nil, fmt.Errorf("failed to load certs: %w", certErr)
	}

	return &Proxy{cert, projectHandler, limiter}, nil
}

func (p *Proxy) ServeHTTP(writer http.ResponseWriter, proxyRequest *http.Request) {
	// parse project and config IDs
	projectID, projectIDOk := proxyRequest.Context().Value("ProjectID").(string)
	if !projectIDOk || !uuidValid(projectID) {
		log.Println("context is missing projectID")
		http.Error(writer, "Unable to process the request", http.StatusInternalServerError)
		return
	}

	configID, configIDOk := proxyRequest.Context().Value("ConfigID").(string)
	if !configIDOk || !uuidValid(configID) {
		log.Println("context is missing configID")
		http.Error(writer, "Unable to process the request", http.StatusInternalServerError)
		return
	}

	// fetch config details
	config, configErr := p.projectHandler.GetConfig(projectID, configID)
	if configErr != nil {
		if configErr == handlers.ErrConfigDoesNotExist {
			log.Println("Config does not exist")
			http.Error(writer, "Config does not exist", http.StatusBadRequest)
			return
		}
		log.Printf("failed to get config: %v\n", configErr)
		http.Error(writer, "Unable to process the request", http.StatusInternalServerError)
		return
	}

	// rate limit
	exceedsRateLimit, rateLimitErr := p.checkExceedsRateLimit(config)
	if rateLimitErr != nil {
		log.Printf("failed to get rate limit: %v\n", rateLimitErr)
		http.Error(writer, "Failed to get rate limit", http.StatusInternalServerError)
		return

	}

	if exceedsRateLimit {
		http.Error(writer, "Too many requests", http.StatusTooManyRequests)
		return
	}

	// create proxy connection
	tlsConn, tlsConnErr := p.createProxyConnection(writer, proxyRequest)
	if tlsConnErr != nil {
		log.Printf("failed to create proxy connection: %v\n", tlsConnErr)
		http.Error(writer, "Unable to process the request", http.StatusInternalServerError)
		return
	}
	defer tlsConn.Close()

	if err := p.handleProxyConnection(tlsConn, proxyRequest.Host, config); err != nil {
		log.Printf("an error has occurred while proxing the connection: %v\n", err)
		tlsConn.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n\r\n"))
		return
	}
}

func (p *Proxy) createProxyConnection(writer http.ResponseWriter, proxyRequest *http.Request) (*tls.Conn, error) {
	if b, err := httputil.DumpRequest(proxyRequest, false); err == nil {
		log.Printf("incoming proxy request:\n%s\n", string(b))
	}
	proxyClient, hijackErr := hijackConnection(writer)
	if hijackErr != nil {
		return nil, hijackErr
	}

	host, _, err := net.SplitHostPort(proxyRequest.Host)
	if err != nil {
		return nil, fmt.Errorf("failed to split host/port: %w", err)
	}

	pemCert, pemKey := createCert([]string{host}, p.cert.Leaf, p.cert.PrivateKey, 240)
	tlsCert, err := tls.X509KeyPair(pemCert, pemKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certs: %w", err)
	}

	if _, err := proxyClient.Write([]byte("HTTP/1.1 200 OK\r\n\r\n")); err != nil {
		return nil, fmt.Errorf("failed to write status to client: %w", err)
	}

	tlsConfig := &tls.Config{
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
		Certificates:     []tls.Certificate{tlsCert},
	}

	return tls.Server(proxyClient, tlsConfig), nil
}

func (p *Proxy) handleProxyConnection(tlsConn *tls.Conn, originalHost string, config *models.Config) error {
	connReader := bufio.NewReader(tlsConn)
	for {
		r, err := http.ReadRequest(connReader)
		if err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("failed to read the request: %w", err)
		}

		if err := p.handleProxyRequest(tlsConn, r, originalHost, config); err != nil {
			return err
		}
	}
	return nil
}

func (p *Proxy) updateRequestHeaders(r *http.Request, config *models.Config) error {
	for _, replacement := range config.HeaderReplacements {
		value, err := DecryptData(replacement.HeaderValue)
		if err != nil {
			return fmt.Errorf("failed to decrypt header replacement: %w", err)
		}
		r.Header.Set(replacement.HeaderName, value)
	}

	return nil
}

func (p *Proxy) handleProxyRequest(tlsConn *tls.Conn, r *http.Request, originalHost string, config *models.Config) error {
	if b, err := httputil.DumpRequest(r, false); err == nil {
		log.Printf("incoming request:\n%s\n", string(b))
	}

	// update request
	if err := setTarget(r, originalHost); err != nil {
		return fmt.Errorf("failed to update request target: %w", err)
	}

	if err := p.updateRequestHeaders(r, config); err != nil {
		return fmt.Errorf("failed to update request headers: %w", err)
	}

	if b, err := httputil.DumpRequest(r, false); err == nil {
		log.Printf("updated request:\n%s\n", string(b))
	}

	var netTransport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 5 * time.Second,
	}

	var netClient = &http.Client{
		Timeout:   time.Second * 30,
		Transport: netTransport,
	}

	// fowrward the request
	log.Println("forwarding request to target...")

	resp, err := netClient.Do(r)
	if err != nil {
		return fmt.Errorf("error forwarding request to the original target: %w", err)
	}

	if b, err := httputil.DumpResponse(resp, false); err == nil {
		log.Printf("target response:\n%s\n", string(b))
	}
	defer resp.Body.Close()

	// send the response back to the client
	if err := resp.Write(tlsConn); err != nil {
		return fmt.Errorf("error forwarding response to the original client: %w", err)
	}

	return nil
}

func (p *Proxy) checkExceedsRateLimit(config *models.Config) (bool, error) {
	limit, limitErr := getLimitForConfig(config)
	if limitErr != nil {
		return false, limitErr
	}

	limitKey := fmt.Sprintf("%s:%s", config.ProjectID, config.ID)
	res, err := p.limiter.Allow(context.Background(), limitKey, limit)
	if err != nil {
		return false, err
	}

	return res.Allowed < 1, nil
}

package proxy

import (
	"api-key-limiter/handlers"
	"api-key-limiter/models"
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"time"
)

type Proxy struct {
	cert           *tls.Certificate
	projectHandler *handlers.ProjectHandler
}

func NewProxy(projectHandler *handlers.ProjectHandler) (*Proxy, error) {
	cert, certErr := loadCA()
	if certErr != nil {
		return nil, fmt.Errorf("failed to load certs: %w", certErr)
	}

	return &Proxy{cert, projectHandler}, nil
}

func (p *Proxy) ServeHTTP(writer http.ResponseWriter, proxyRequest *http.Request) {
	projectID, ok := proxyRequest.Context().Value("ProjectID").(string)
	if !ok || projectID == "" {
		log.Println("context is missing projectID")
		http.Error(writer, "Unable to process the request", http.StatusInternalServerError)
		return
	}

	configID := "88bbcc17-096a-40fb-9b32-b519ad834cea"
	config, configErr := p.projectHandler.GetConfig(projectID, configID)

	if configErr != nil {
		if configErr == handlers.ErrConfigDoesNotExist {
			http.Error(writer, "Config does not exist", http.StatusBadRequest)
			return
		}
		log.Printf("failed to get config: %w\n", configErr)
		http.Error(writer, "Unable to process the request", http.StatusInternalServerError)
		return
	}

	tlsConn, tlsConnErr := p.createProxyConnection(writer, proxyRequest)
	if tlsConnErr != nil {
		log.Printf("failed to create proxy connection: %w\n", tlsConnErr)
		http.Error(writer, "Unable to process the request", http.StatusInternalServerError)
		return
	}
	defer tlsConn.Close()

	if err := p.handleProxyConnection(tlsConn, proxyRequest.Host, config); err != nil {
		log.Printf("an error has occurred while proxing the connection: %w\n", err)
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

func (p *Proxy) handleProxyRequest(tlsConn *tls.Conn, r *http.Request, originalHost string, config *models.Config) error {
	if b, err := httputil.DumpRequest(r, false); err == nil {
		log.Printf("incoming request:\n%s\n", string(b))
	}

	// update request
	if err := setTarget(r, originalHost); err != nil {
		return fmt.Errorf("failed to update request target", err)
	}

	r.Header.Set(config.HeaderName, config.HeaderValue)

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

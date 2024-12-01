package proxy

import (
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
	cert *tls.Certificate
}

func NewProxy() (*Proxy, error) {
	cert, certErr := loadCA()
	if certErr != nil {
		return nil, fmt.Errorf("failed to load certs: %w", certErr)
	}

	return &Proxy{cert}, nil
}

func (p *Proxy) ServeHTTP(writer http.ResponseWriter, proxyRequest *http.Request) {
	tlsConn, tlsConnErr := p.createProxyConnection(writer, proxyRequest)
	if tlsConnErr != nil {
		log.Printf("failed to create proxy connection: %w", tlsConnErr)
		// TODO: write error response to client
		return
	}
	defer tlsConn.Close()

	if err := p.handleProxyConnection(tlsConn, proxyRequest.Host); err != nil {
		log.Printf("an error has occured while proxing the connection: %w", err)
		// TODO: write error response to tlsConn
		return
	}
}

func (p *Proxy) createProxyConnection(writer http.ResponseWriter, proxyRequest *http.Request) (*tls.Conn, error) {
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

func (p *Proxy) handleProxyConnection(tlsConn *tls.Conn, originalHost string) error {
	connReader := bufio.NewReader(tlsConn)
	for {
		r, err := http.ReadRequest(connReader)
		if err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("failed to read the request: %w", err)
		}

		// read client request
		if b, err := httputil.DumpRequest(r, false); err == nil {
			log.Printf("incoming request:\n%s\n", string(b))
		}

		// fowrward the request
		if err := setTarget(r, originalHost); err != nil {
			return fmt.Errorf("failed to update request target", err)
		}

		log.Println("forwarding request to target...")

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

	}
	return nil
}

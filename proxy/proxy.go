package proxy

import (
	"bufio"
	"crypto/tls"
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

func NewProxy() *Proxy {
	cert, certErr := loadCA()
	if certErr != nil {
		log.Fatalf("failed to load certs: %v\n", certErr)
	}

	return &Proxy{cert}
}

func (p *Proxy) ServeHTTP(writer http.ResponseWriter, proxyRequest *http.Request) {
	hij, ok := writer.(http.Hijacker)
	if !ok {
		log.Fatal("cannot convert connection to hijacker")
	}

	proxyClient, _, e := hij.Hijack()
	if e != nil {
		panic("cannot hijack connection " + e.Error())
	}

	host, _, err := net.SplitHostPort(proxyRequest.Host)
	if err != nil {
		log.Fatal("error splitting host/port:", err)
	}

	pemCert, pemKey := createCert([]string{host}, p.cert.Leaf, p.cert.PrivateKey, 240)
	tlsCert, err := tls.X509KeyPair(pemCert, pemKey)
	if err != nil {
		log.Fatal("failed to create cert:", err)
	}

	if _, err := proxyClient.Write([]byte("HTTP/1.1 200 OK\r\n\r\n")); err != nil {
		log.Fatal("error writing status to client:", err)
	}

	tlsConfig := &tls.Config{
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
		Certificates:     []tls.Certificate{tlsCert},
	}

	tlsConn := tls.Server(proxyClient, tlsConfig)
	defer tlsConn.Close()
	p.handleProxyConnection(tlsConn, proxyRequest.Host)

}

func (p *Proxy) handleProxyConnection(tlsConn *tls.Conn, originalHost string) {
	connReader := bufio.NewReader(tlsConn)
	for {
		r, err := http.ReadRequest(connReader)
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatal("request reader err", err)
		}
		// We can dump the request; log it, modify it...
		if b, err := httputil.DumpRequest(r, false); err == nil {
			log.Printf("incoming request:\n%s\n", string(b))
		}

		if err := setTarget(r, originalHost); err != nil {
			log.Fatalf("failed to update request target: %w\n", err)
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
			log.Fatal("error sending request to target:", err)
		}

		if b, err := httputil.DumpResponse(resp, false); err == nil {
			log.Printf("target response:\n%s\n", string(b))
		}
		defer resp.Body.Close()

		// send the response back to the client
		if err := resp.Write(tlsConn); err != nil {
			log.Println("error writing response back:", err)
		}
	}
}

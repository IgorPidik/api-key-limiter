package proxy

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

func hijackConnection(writer http.ResponseWriter) (net.Conn, error) {
	hij, ok := writer.(http.Hijacker)
	if !ok {
		return nil, ErrFailedToConvertConnectionToHijacker
	}

	proxyClient, _, err := hij.Hijack()
	if err != nil {
		return nil, err
	}

	return proxyClient, nil
}

func setTarget(req *http.Request, targetHost string) error {
	if !strings.HasPrefix(targetHost, "https") {
		targetHost = "https://" + targetHost
	}

	targetUrl, err := url.Parse(targetHost)
	if err != nil {
		return err
	}

	targetUrl.Path = req.URL.Path
	targetUrl.RawQuery = req.URL.RawQuery
	req.URL = targetUrl
	req.RequestURI = ""
	return nil
}

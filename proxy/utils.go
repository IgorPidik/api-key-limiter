package proxy

import (
	"net/http"
	"net/url"
	"strings"
)

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

package proxy

import (
	"api-key-limiter/models"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-redis/redis_rate/v10"
	"github.com/google/uuid"
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

func uuidValid(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}

func getLimitForConfig(config *models.Config) (redis_rate.Limit, error) {
	switch config.LimitPer {
	case "second":
		return redis_rate.PerSecond(config.LimitNumberOfRequests), nil
	case "minute":
		return redis_rate.PerMinute(config.LimitNumberOfRequests), nil
	case "hour":
		return redis_rate.PerHour(config.LimitNumberOfRequests), nil
	case "day":
		return redis_rate.Limit{
			Rate:   config.LimitNumberOfRequests,
			Burst:  config.LimitNumberOfRequests,
			Period: time.Hour * 24,
		}, nil
	default:
		return redis_rate.PerSecond(0), fmt.Errorf("unsupported limit per unit: %s", config.LimitPer)
	}
}

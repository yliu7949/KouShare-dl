package proxy

import (
	"log"
	"net/http"
	"net/url"
	"time"
)

var Client = http.Client{
	Transport: defaultTransport(http.ProxyFromEnvironment),
}

func EnableProxy(proxyURL string) {
	proxyFunc := http.ProxyFromEnvironment
	if proxyURL != "" {
		u, err := url.Parse(proxyURL)
		if err != nil {
			log.Fatal("Parse proxy url error: ", err)
		}
		proxyFunc = http.ProxyURL(u)
	}

	Client = http.Client{
		Transport: defaultTransport(proxyFunc),
	}
}

func defaultTransport(proxyFunc func(*http.Request) (*url.URL, error)) *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = proxyFunc
	transport.ResponseHeaderTimeout = 30 * time.Second
	transport.IdleConnTimeout = 90 * time.Second
	return transport
}

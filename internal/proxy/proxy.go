package proxy

import (
	"crypto/tls"
	"log"
	"net/http"
	"net/url"
)

var Client = http.Client{}

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
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			Proxy: proxyFunc,
		},
	}
}

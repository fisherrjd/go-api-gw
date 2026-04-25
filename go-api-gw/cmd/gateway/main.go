package main

import (
	"log"
	"net/http"
	"net/url"
	"time"
	"github.com/fisherrjd/go-api-gw/pkg/proxy"

)

func main() {

	target, err := url.Parse("https://httpbin.org/")
	if err != nil {
        log.Fatal(err)
    }

    client:= &http.Client{
    	Timeout: 30 * time.Second,
    }

    p := &proxy.Proxy{
        Target: target,
        Client: client,
    }

	http.Handle("/", p)
	log.Fatal(http.ListenAndServe(":30420", nil))
}

package main

import (
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/fisherrjd/go-api-gw/pkg/proxy"
	"github.com/fisherrjd/go-api-gw/pkg/router"
)

func main() {

	tv1, err := url.Parse("http://localhost:8001")
	tv3, err := url.Parse("http://localhost:8002")
	tv4, err := url.Parse("http://localhost:8003")

	if err != nil {
		log.Fatal(err)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	v1 := &proxy.Proxy{
		Target: tv1,
		Client: client,
	}

	v3 := &proxy.Proxy{
		Target: tv3,
		Client: client,
	}

	v4 := &proxy.Proxy{
		Target: tv4,
		Client: client,
	}

	rt := map[string]http.Handler{
		"/v1": v1,
		"/v3": v3,
		"/v4": v4,
	}

	r := router.NewRouter(rt)
	http.Handle("/", r)

	log.Fatal(http.ListenAndServe(":30420", nil))
}

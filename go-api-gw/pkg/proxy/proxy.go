package proxy

import (
	"io"
	"net/http"
	"net/url"
)

type Proxy struct {
	Target *url.URL
	Client *http.Client
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fullURL := p.Target.String() + r.URL.Path
	if r.URL.RawQuery != "" {
		fullURL = fullURL + "?" + r.URL.RawQuery
	}
	outReq, err := http.NewRequestWithContext(r.Context(), r.Method, fullURL, r.Body)
	outReq.Host = p.Target.Host
	resp, err := p.Client.Do(outReq)
	if err != nil {
		http.Error(w, "upstream failed", http.StatusBadGateway)
		return
	}
	io.Copy(w, resp.Body)
	defer resp.Body.Close()
}

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
	statusCode := 0

	fullURL := p.Target.String() + r.URL.Path
	if r.URL.RawQuery != "" {
		fullURL = fullURL + "?" + r.URL.RawQuery
	}
	outReq, err := http.NewRequestWithContext(r.Context(), r.Method, fullURL, r.Body)

	if err != nil {
		statusCode = http.StatusInternalServerError
		http.Error(w, "Failed to create upstream quest", statusCode)
		return
	}
	outReq.Header = r.Header.Clone()
	outReq.Host = p.Target.Host
	resp, err := p.Client.Do(outReq)

	for key, values := range resp.Header {
		for _, v := range values {
			w.Header().Add(key, v)
		}
	}

	statusCode = resp.StatusCode
	w.WriteHeader(statusCode)

	if err != nil {
		statusCode = http.StatusInternalServerError
		http.Error(w, "upstream failed", statusCode)
		return
	}
	defer resp.Body.Close()

	io.Copy(w, resp.Body)
}

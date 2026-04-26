package proxy

import (
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Proxy struct {
	Target *url.URL
	Client *http.Client
}

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	statusCode := 0
	apiKey :=""
	defer func() {
		logger.Info("request", "method", r.Method, "duration", time.Since(start).Milliseconds(), "request_id", r.Header.Get("X-Request-ID"), "path", r.URL.Path, "api_key", apiKey, "status", statusCode)
	}()
	r = stripForwardedHeaders(r)
	r, requestID := addRequestID(r)
	r, clientIP := addForwardedFor(r)

	if APIKeyEmpty(r) {
		statusCode = http.StatusUnauthorized
		http.Error(w, "Missing X-API-Key", statusCode)
		return
	}

	if APIKeyInvalid(r) {
		statusCode = http.StatusUnauthorized
		http.Error(w, "Invalid X-API-Key", statusCode)
		return
	}

	apiKey = r.Header.Get("X-API-Key")
	r.Header.Del("X-API-Key")

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
	if err != nil {
		statusCode = http.StatusInternalServerError
		http.Error(w, "upstream failed", statusCode)
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, v := range values {
			w.Header().Add(key, v)
		}
	}

	// 2. Add your request ID to response (so client can see it)
	w.Header().Add("X-Request-Id", requestID)   // You need to capture this from earlier!
	w.Header().Add("X-Forwarded-For", clientIP) // You need to capture this from earlier!

	// 3. Status code
	statusCode = resp.StatusCode
	w.WriteHeader(statusCode)
	io.Copy(w, resp.Body)
}

func stripForwardedHeaders(r *http.Request) *http.Request {
	// Iterate over all headers
	for key, _ := range r.Header {
		// Check if key matches pattern
		if strings.HasPrefix(key, "X-Forwarded-") {
			// Delete it
			r.Header.Del(key)
		}
	}
	return r
}

func addRequestID(r *http.Request) (*http.Request, string) {
	requestID := uuid.New().String()
	r.Header.Add("X-Request-ID", requestID)

	return r, requestID
}

func addForwardedFor(r *http.Request) (*http.Request, string) {
	clientIP := strings.Split(r.RemoteAddr, ":")[0]
	r.Header.Add("X-Forwarded-For", clientIP)
	return r, clientIP
}

func APIKeyEmpty(r *http.Request) bool {
	return r.Header.Get("X-API-Key") == ""
}

func APIKeyInvalid(r *http.Request) bool {
	return r.Header.Get("X-API-Key") != "valid-key-123"
}

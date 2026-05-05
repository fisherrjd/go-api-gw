package router

import (
	"net/http"
	"strings"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/fisherrjd/go-api-gw/pkg/ratelimit"


)

type statusRecorder struct {
    http.ResponseWriter
    status int
}

func (sr *statusRecorder) WriteHeader(code int) {
    sr.status = code
    sr.ResponseWriter.WriteHeader(code)
}


type Router struct {
	routes map[string]http.Handler
}

func NewRouter(rts map[string]http.Handler) *Router {
	return &Router{routes: rts}
}

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	statusCode := 0
	apiKey := ""
	outReq := r.Clone(r.Context())
	sr := &statusRecorder{ResponseWriter: w}

	defer func() {
		logger.Info("request", "method", r.Method, "duration", time.Since(start).Milliseconds(), "request_id", outReq.Header.Get("X-Request-ID"), "path", outReq.URL.Path, "api_key", apiKey, "status", statusCode)
	}()



	outReq = stripForwardedHeaders(outReq)
	outReq, requestID := addRequestID(outReq)
	outReq, clientIP := addForwardedFor(outReq)

	if APIKeyEmpty(outReq) {
		statusCode = http.StatusUnauthorized
		http.Error(w, "Missing X-API-Key", statusCode)
		return
	}

	if APIKeyInvalid(outReq) {
		statusCode = http.StatusUnauthorized
		http.Error(w, "Invalid X-API-Key", statusCode)
		return
	}

	apiKey = outReq.Header.Get("X-API-Key")
	outReq.Header.Del("X-API-Key")


	// 2. Add your request ID to response (so client can see it)
	w.Header().Add("X-Request-Id", requestID)   // You need to capture this from earlier!
	w.Header().Add("X-Forwarded-For", clientIP) // You need to capture this from earlier!

	for prefix, handler := range rt.routes {
		if strings.HasPrefix(outReq.URL.Path, prefix) {
			outPath := strings.TrimPrefix(outReq.URL.Path, prefix)
			outReq.URL.Path = outPath
			handler.ServeHTTP(sr, outReq)
			statusCode = sr.status
			return
		}
	}

	statusCode = http.StatusNotFound
	http.Error(w, "Route not found", statusCode)
	return
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

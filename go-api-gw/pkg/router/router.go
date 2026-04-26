package router

import (
	"net/http"
	"strings"
)

type Router struct {
	routes map[string]http.Handler
}

func NewRouter(rts map[string]http.Handler) *Router {
	return &Router{routes: rts}
}

func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	outReq := r.Clone(r.Context())

	for prefix, handler := range rt.routes {
		if strings.HasPrefix(outReq.URL.Path, prefix) {
			outPath := strings.TrimPrefix(outReq.URL.Path, prefix)
			outReq.URL.Path = outPath
			handler.ServeHTTP(w, outReq)
			return
		}
	}

	statusCode := http.StatusNotFound
	http.Error(w, "Route not found", statusCode)
	return
}

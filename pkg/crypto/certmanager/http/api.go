/*
Package http implements a http-01 ACME protocol responder.
*/
package http

import (
	"net/http"
	"sync"

	"github.com/Cloud-Foundations/golib/pkg/log"
)

type RedirectHandler struct{}

// ServeHTTP serves HTTP requests. All GET and HEAD requests will be redirected
// to the default TLS port 443 with 302 Found status code, preserving the
// original request path and query. It responds with 400 Bad Request to all
// other HTTP methods.
func (h *RedirectHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h.serveHTTP(w, req)
}

type Responder struct {
	fallback  http.Handler
	logger    log.DebugLogger
	rwMutex   sync.RWMutex // Protect everything below.
	responses map[string]string
}

// CreateRedirectServer is a convenience function that creates a redirecting
// HTTP server on port portNum. Do not use this if NewServer is also used.
func CreateRedirectServer(portNum uint16, logger log.DebugLogger) error {
	return createRedirectServer(portNum, logger)
}

// NewHandler creates a HTTP handler. The handler will respond to ACME
// "http-01" challenges.
// A *RedirectHandler may be passed in through fallback. If this is nil, all
// non-ACME challenge responses will be rejected.
// The logger is used for logging messages.
func NewHandler(fallback http.Handler,
	logger log.DebugLogger) (*Responder, error) {
	return newHandler(fallback, logger)
}

// NewServer creates a HTTP server on port number portNum. This should
// normally be 80, unless your firewall is configured to DNAT from port 80 on
// a public IP to portNum.
// The server will respond to ACME "http-01" challenges.
// A *RedirectHandler may be passed in through fallback. If this is nil, a call
// to the Cleanup method will cause new connections to be accepted and
// immediately closed until the next Respond method is called, minimising the
// attack surface.
// The logger is used for logging messages.
func NewServer(portNum uint16, fallback http.Handler,
	logger log.DebugLogger) (*Responder, error) {
	return newServer(portNum, fallback, logger)
}

func (r *Responder) Cleanup() {
	r.cleanup()
}

func (r *Responder) Respond(key, value string) error {
	return r.respond(key, value)
}

func (r *Responder) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.serveHTTP(w, req)
}

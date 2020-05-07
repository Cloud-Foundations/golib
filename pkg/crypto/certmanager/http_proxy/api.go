/*
Package http_proxy implements a http-01 ACME protocol responder using the
acme-proxy.
*/
package http_proxy

import (
	"github.com/Cloud-Foundations/golib/pkg/log"
)

type Responder struct {
	acmeProxy string
	logger    log.DebugLogger
}

// New creates a *Responder for ACME "http-01" challenges.
// The address of the acme-proxy must be given by acmeProxy.
// The logger is used for logging messages.
func New(acmeProxy string, logger log.DebugLogger) (*Responder, error) {
	return newResponder(acmeProxy, logger)
}

func (r *Responder) Cleanup() {
	r.cleanup()
}

func (r *Responder) Respond(key, value string) error {
	return r.respond(key, value)
}

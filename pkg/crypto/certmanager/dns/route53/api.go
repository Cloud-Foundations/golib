/*
Package route53 implements a dns-01 ACME protocol responder using AWS Route 53.
*/
package route53

import (
	"time"

	"github.com/Cloud-Foundations/golib/pkg/log"
)

type recordDeleteWriter interface {
	DeleteRecords(fqdn, recType string) error
	WriteRecords(fqdn, recType string, recs []string, ttl time.Duration) error
}

type Responder struct {
	rdw    recordDeleteWriter
	logger log.DebugLogger
	// Mutable data follow.
	records map[string]string
}

// New creates a DNS responder for ACME dns-01 challenges.
// The logger is used for logging messages.
func New(hostedZoneId string,
	logger log.DebugLogger) (*Responder, error) {
	return newResponder(hostedZoneId, logger)
}

func (r *Responder) Cleanup() {
	r.cleanup()
}

func (r *Responder) Respond(key, value string) error {
	return r.respond(key, value)
}

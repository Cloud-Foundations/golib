/*
Package route53 implements a dns-01 ACME protocol responder using AWS Route 53.
*/
package route53

import (
	"github.com/Cloud-Foundations/golib/pkg/log"
	"github.com/aws/aws-sdk-go/service/route53"
)

type Responder struct {
	awsService   *route53.Route53
	hostedZoneId *string
	logger       log.DebugLogger
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

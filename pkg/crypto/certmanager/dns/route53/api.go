/*
Package route53 implements a dns-01 ACME protocol responder using AWS Route 53.
*/
package route53

import (
	"github.com/Cloud-Foundations/golib/pkg/crypto/certmanager"
	"github.com/Cloud-Foundations/golib/pkg/log"
)

// New creates a DNS responder for ACME dns-01 challenges.
// The logger is used for logging messages.
func New(hostedZoneId string,
	logger log.DebugLogger) (certmanager.Responder, error) {
	return newResponder(hostedZoneId, logger)
}

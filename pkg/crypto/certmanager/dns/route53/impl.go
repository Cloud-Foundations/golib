package route53

import (
	"errors"

	"github.com/Cloud-Foundations/golib/pkg/crypto/certmanager"
	"github.com/Cloud-Foundations/golib/pkg/dns/route53"
	"github.com/Cloud-Foundations/golib/pkg/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
)

func newResponder(hostedZoneId string,
	logger log.DebugLogger) (certmanager.Responder, error) {
	if hostedZoneId == "" {
		return nil, errors.New("no hosted zone ID specified")
	}
	awsSession, err := session.NewSession(&aws.Config{})
	if err != nil {
		return nil, err
	}
	if awsSession == nil {
		return nil, errors.New("awsSession == nil")
	}
	rdw, err := route53.New(awsSession, hostedZoneId, logger)
	if err != nil {
		return nil, err
	}
	return certmanager.MakeDnsResponder(rdw, logger)
}

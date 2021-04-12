package route53

import (
	"errors"
	"time"

	"github.com/Cloud-Foundations/golib/pkg/dns/route53"
	"github.com/Cloud-Foundations/golib/pkg/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
)

func newResponder(hostedZoneId string,
	logger log.DebugLogger) (*Responder, error) {
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
	return &Responder{
		rdw:     rdw,
		logger:  logger,
		records: make(map[string]string),
	}, nil
}

func (r *Responder) cleanup() {
	if len(r.records) < 1 {
		return
	}
	for fqdn := range r.records {
		if err := r.rdw.DeleteRecords(fqdn, "TXT"); err != nil {
			r.logger.Println(err)
		} else {
			delete(r.records, fqdn)
		}
	}
}

func (r *Responder) respond(key, value string) error {
	if r.records[key] == value {
		return nil
	}
	r.logger.Debugf(1, "publishing %s TXT=\"%s\"\n", key, value)
	err := r.rdw.WriteRecords(key, "TXT", []string{value}, time.Second*15)
	if err != nil {
		return err
	}
	r.records[key] = value
	return nil
}

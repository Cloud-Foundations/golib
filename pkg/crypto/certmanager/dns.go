package certmanager

import (
	"golang.org/x/crypto/acme"
	"time"

	"github.com/Cloud-Foundations/golib/pkg/dns"
	"github.com/Cloud-Foundations/golib/pkg/log"
)

type dnsResponder struct {
	rdw    dns.RecordDeleteWriter
	logger log.DebugLogger
	// Mutable data follow.
	records map[string]string
}

func (cm *CertificateManager) respondDNS(domain string,
	challenge *acme.Challenge) error {
	response, err := cm.acmeClient.DNS01ChallengeRecord(challenge.Token)
	if err != nil {
		return err
	}
	return cm.responder.Respond("_acme-challenge."+domain, response)
}

func makeDnsResponder(rdw dns.RecordDeleteWriter,
	logger log.DebugLogger) (Responder, error) {
	return &dnsResponder{
		rdw:     rdw,
		logger:  logger,
		records: make(map[string]string),
	}, nil
}

func (r *dnsResponder) Cleanup() {
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

func (r *dnsResponder) Respond(key, value string) error {
	if r.records[key] == value {
		return nil
	}
	r.logger.Debugf(1, "publishing %s TXT=\"%s\"\n", key, value)
	err := r.rdw.WriteRecords(key, "TXT", []string{value}, time.Second*15, true)
	if err != nil {
		return err
	}
	r.records[key] = value
	return nil
}

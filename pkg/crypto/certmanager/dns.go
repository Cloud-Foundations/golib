package certmanager

import (
	"golang.org/x/crypto/acme"
)

func (cm *CertificateManager) respondDNS(domain string,
	challenge *acme.Challenge) error {
	response, err := cm.acmeClient.DNS01ChallengeRecord(challenge.Token)
	if err != nil {
		return err
	}
	return cm.responder.Respond("_acme-challenge."+domain, response)
}

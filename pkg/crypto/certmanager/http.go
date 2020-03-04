package certmanager

import (
	"golang.org/x/crypto/acme"
)

func (cm *CertificateManager) respondHTTP(challenge *acme.Challenge) error {
	response, err := cm.acmeClient.HTTP01ChallengeResponse(challenge.Token)
	if err != nil {
		return err
	}
	return cm.responder.Respond(
		cm.acmeClient.HTTP01ChallengePath(challenge.Token), response)
}

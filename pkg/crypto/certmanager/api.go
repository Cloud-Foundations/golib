/*
Package certmanager uses the ACME protocol to request and automatically renew
X509 certificates. It supports concurrency and sharing across server instances.

This package wraps the ACME protocol so that the application has easy access to
certficates signed by a public Certificate Authority (CA) such as Let's Encrypt.
Multiple server instances can safely use this package at the same time to
request certificates by providing a Locker.

Certificate sharing may be enabled by providing a Storer, which may be required
to avoid duplicate certificate limits imposed by the CA. Duplicate certificate
requests typically occur if there are multiple server instances or when
instances are redeployed.

The aws package implements a Locker and Storer using AWS Secrets manager and a
DNS-based Responder using Route53.

The http package implements a HTTP-based Responder.
*/
package certmanager

import (
	"crypto"
	"crypto/tls"
	"sync"
	"time"

	"golang.org/x/crypto/acme"

	"github.com/Cloud-Foundations/golib/pkg/log"
)

const LetsEncryptProductionURL = acme.LetsEncryptURL
const LetsEncryptStagingURL = "https://acme-staging-v02.api.letsencrypt.org/directory"

type Certificate struct {
	CertPemBlock []byte
	KeyPemBlock  []byte
	tlsCert      tls.Certificate
	notAfter     time.Time
	notBefore    time.Time
}

type CertificateManager struct {
	acmeClient     *acme.Client // Only used in the renewal goroutine.
	acmeOrder      *acme.Order  // Only used in the renewal goroutine.
	caDirectoryURL string
	certFilename   string
	challengeType  string
	keyFilename    string
	key            crypto.Signer
	locker         Locker
	names          []string
	renewBefore    float64
	responder      Responder
	storer         Storer
	logger         log.DebugLogger
	writeNotifier  chan struct{}
	rwMutex        sync.RWMutex // Protect everything below.
	certificate    *Certificate
}

// Locker is an interface to a remote locking mechanism.
type Locker interface {
	// GetLostChannel returns a channel where notifications are sent if the lock
	// is lost (such as due to a transaction timeout). This may return nil.
	GetLostChannel() <-chan error

	// Lock attempts to grab the lock, blocking until ready or error.
	Lock() error

	// Unlock releases the lock. It may return an error if the lock was broken
	// or for other reasons.
	Unlock() error
}

// Responder implements a challenge responder. Typical implementations would be
// either a DNS TXT record responder (key=FQDN) for the "dns-01" challenge or a
// HTTP responder (key=path) for the "http-01" challenge.
type Responder interface {
	Cleanup()
	Respond(key, value string) error
}

// Storer is an interface to a remote data store.
type Storer interface {
	// Read will read arbitrary data from the remote store.
	Read() (*Certificate, error)

	// Write will write arbitrary data to the remote store.
	Write(cert *Certificate) error
}

// New creates a *CertificateManager for the domain(s) listed in names.
// The certificate and private key are cached locally in the files named by
// certFilename and keyFilename. If either is empty then no local cache is
// employed.
// The locker is used to ensure only one ACME transaction is performed at any
// time. If this is nil, no transaction locking is performed.
// The type of challenge to use is specified by challengeType. Currently
// "dns-01" and "http-01" are supported.
// The storer is used to store the certificate and private key for sharing with
// other instances of the service. If this is nil, no sharing is performed.
// Certificates are renewedBefore expiration, specified as a fraction of the
// certificate lifetime. For example, if the CA issues certificates with a
// lifetime of 90 days, a value of 0.33 will cause certificates to be renewed
// 29.7 days prior to expiration. If 0, the default is a random value between
// 0.32 and 0.34 (roughly 30 days for a 90 day certificate).
// Renewals will not be attempted more than once per hour.
// The responder is used to respond to ACME challenges.
// The Certificate Authority directory endpoint is specified by caDirectoryURL.
// If this is the empty string, Let's Encrypt (Production) is used.
// The logger is used for logging messages.
// Background work will be scheduled to renew the certificate.
func New(names []string, certFilename, keyFilename string, locker Locker,
	challengeType string, responder Responder, storer Storer,
	renewBefore float64, caDirectoryURL string,
	logger log.DebugLogger) (*CertificateManager, error) {
	return newManager(names, certFilename, keyFilename, locker, challengeType,
		responder, storer, renewBefore, caDirectoryURL, logger)
}

// GetCertificate yields the most recently renewed certificate. The method
// value may be assigned to the crypto/tls.Config.GetCertificate field.
func (cm *CertificateManager) GetCertificate(hello *tls.ClientHelloInfo) (
	*tls.Certificate, error) {
	return cm.getCertificate(hello)
}

// GetWriteNotifier returns the channel to which certificate write notifications
// are sent.
func (cm *CertificateManager) GetWriteNotifier() <-chan struct{} {
	return cm.writeNotifier
}

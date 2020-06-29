package certmanager

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/acme"

	"github.com/Cloud-Foundations/Dominator/lib/format"
	"github.com/Cloud-Foundations/golib/pkg/log"
)

const defaultRsaKeySize = 2048

var supportedChallengeTypes = map[string]struct{}{
	"dns-01":  {},
	"http-01": {},
}

func loadCertificate(certFilename, keyFilename string,
	logger log.Logger) (*Certificate, error) {
	certPemBlock, err := ioutil.ReadFile(certFilename)
	if err != nil {
		return nil, err
	}
	keyPemBlock, err := ioutil.ReadFile(keyFilename)
	if err != nil {
		return nil, err
	}
	cert := &Certificate{CertPemBlock: certPemBlock, KeyPemBlock: keyPemBlock}
	if err := cert.parse(); err != nil {
		return nil, err
	}
	logger.Printf("loaded certificate from: %s, expires on: %s (in: %s)\n",
		certFilename, cert.notAfter.Local(),
		format.Duration(time.Until(cert.notAfter)))
	return cert, nil
}

func makeKeyECDSA() (crypto.Signer, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

func makeKeyRSA() (crypto.Signer, error) {
	return rsa.GenerateKey(rand.Reader, defaultRsaKeySize)
}

func readCert(storer Storer) (*Certificate, error) {
	if cert, err := storer.Read(); err != nil {
		return nil, err
	} else if err := cert.parse(); err != nil {
		return nil, err
	} else {
		return cert, nil
	}
}

func makeCert(chainDER [][]byte, key crypto.Signer) (*Certificate, error) {
	if len(chainDER) < 1 {
		return nil, errors.New("empty chain")
	}
	leaf, err := x509.ParseCertificate(chainDER[0])
	if err != nil {
		return nil, err
	}
	certPemBlock := &bytes.Buffer{}
	for index, certDER := range chainDER {
		if index > 0 {
			fmt.Fprintln(certPemBlock)
		}
		err := pem.Encode(certPemBlock, &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: certDER,
		})
		if err != nil {
			return nil, err
		}
	}
	keyPemBlock := &bytes.Buffer{}
	switch key := key.(type) {
	case *ecdsa.PrivateKey:
		keyDER, err := x509.MarshalECPrivateKey(key)
		if err != nil {
			return nil, err
		}
		err = pem.Encode(keyPemBlock, &pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: keyDER,
		})
		if err != nil {
			return nil, err
		}
	case *rsa.PrivateKey:
		keyDER := x509.MarshalPKCS1PrivateKey(key)
		err := pem.Encode(keyPemBlock, &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: keyDER,
		})
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unsupported private key type")
	}
	cert := &Certificate{
		CertPemBlock: certPemBlock.Bytes(),
		KeyPemBlock:  keyPemBlock.Bytes(),
		tlsCert: tls.Certificate{
			Certificate: chainDER,
			PrivateKey:  key,
			Leaf:        leaf,
		},
		notAfter:  leaf.NotAfter,
		notBefore: leaf.NotBefore,
	}
	return cert, nil
}

func jitteryHour() time.Duration {
	randByte := make([]byte, 1)
	rand.Read(randByte)
	return time.Hour + time.Second*time.Duration(randByte[0])
}

func (cert *Certificate) parse() error {
	var err error
	cert.tlsCert, err = tls.X509KeyPair(cert.CertPemBlock, cert.KeyPemBlock)
	if err != nil {
		return err
	}
	x509Cert, err := x509.ParseCertificate(cert.tlsCert.Certificate[0])
	if err != nil {
		return err
	}
	cert.tlsCert.Leaf = x509Cert
	cert.notAfter = x509Cert.NotAfter
	cert.notBefore = x509Cert.NotBefore
	return nil
}

func (cert *Certificate) timeUntilRenewal(renewBefore float64) time.Duration {
	if cert == nil {
		return -1
	}
	lifetime := cert.notAfter.Sub(cert.notBefore)
	if lifetime < time.Hour {
		lifetime = jitteryHour()
	}
	return time.Until(cert.notAfter.Add(
		-time.Duration(lifetime.Seconds()*renewBefore) * time.Second))
}

func newManager(names []string, certFilename, keyFilename string, locker Locker,
	challengeType string, responder Responder, storer Storer,
	renewBefore float64, caDirectoryURL, keyType string,
	logger log.DebugLogger) (*CertificateManager, error) {
	if challengeType == "" {
		cert, err := loadCertificate(certFilename, keyFilename, logger)
		if err != nil {
			return nil, err
		}
		return &CertificateManager{certificate: cert}, nil
	}
	if locker == nil {
		locker = nullLocker{}
	}
	if _, ok := supportedChallengeTypes[challengeType]; !ok {
		return nil,
			fmt.Errorf("challenge type: %s not supported", challengeType)
	}
	if renewBefore <= 0.0 {
		randByte := make([]byte, 1)
		if _, err := rand.Read(randByte); err != nil {
			return nil, err
		}
		// Compute random number between 0.32 and 0.34.
		renewBefore = 0.32 + 0.02*float64(randByte[0])/256.0
	}
	if responder == nil {
		return nil, errors.New("no responder specified")
	}
	keyMaker := makeKeyECDSA
	switch keyType {
	case "", "EC":
	case "RSA":
		keyMaker = makeKeyRSA
	default:
		return nil, errors.New("unsupported key type: " + keyType)
	}
	cm := &CertificateManager{
		caDirectoryURL: caDirectoryURL,
		certFilename:   certFilename,
		challengeType:  challengeType,
		keyFilename:    keyFilename,
		keyMaker:       keyMaker,
		locker:         locker,
		names:          names,
		renewBefore:    renewBefore,
		responder:      responder,
		storer:         storer,
		logger:         logger,
		writeNotifier:  make(chan struct{}, 1),
	}
	go cm.begin()
	return cm, nil
}

func (cm *CertificateManager) authorise(ctx context.Context,
	authoriseUrl string) error {
	authorisation, err := cm.acmeClient.GetAuthorization(ctx, authoriseUrl)
	if err != nil {
		return err
	}
	if authorisation.Status != acme.StatusPending {
		return nil
	}
	var challenge *acme.Challenge
	for _, chal := range authorisation.Challenges {
		if chal.Type == cm.challengeType {
			challenge = chal
			break
		}
	}
	if challenge == nil {
		return fmt.Errorf(
			"unable to satisfy %s for domain %s: no viable challenge type found",
			authorisation.URI, authorisation.Identifier.Value)
	}
	domain := authorisation.Identifier.Value
	switch cm.challengeType {
	case "dns-01":
		if err := cm.respondDNS(domain, challenge); err != nil {
			return err
		}
	case "http-01":
		if err := cm.respondHTTP(challenge); err != nil {
			return err
		}
	default:
		return errors.New("unknown challenge type")
	}
	_, err = cm.acmeClient.Accept(ctx, challenge)
	if err != nil {
		return err
	}
	_, err = cm.acmeClient.WaitAuthorization(ctx, authorisation.URI)
	return err
}

func (cm *CertificateManager) begin() {
	if err := cm.fileLoad(); err != nil {
		cm.logger.Println(err)
	}
	for {
		wait := cm.checkRenew()
		cm.logger.Printf(
			"scheduling next certificate renewal check at: %s (in: %s)\n",
			time.Now().Add(wait), format.Duration(wait))
		time.Sleep(wait)
	}
}

func (cm *CertificateManager) checkRenew() time.Duration {
	cm.rwMutex.RLock()
	cert := cm.certificate
	cm.rwMutex.RUnlock()
	// First check if the current certificate needs to be renewed.
	if cert != nil {
		if interval := cert.timeUntilRenewal(cm.renewBefore); interval > 0 {
			return interval
		}
	}
	if cm.storer != nil {
		// Now try and get a certifcate from the remote store, save it and see
		// if it needs to be renewed.
		if cert, err := readCert(cm.storer); err != nil {
			cm.logger.Println(err)
		} else { // Make use of newer certificate, even if expired, then rewnew.
			cm.rwMutex.Lock()
			if cm.certificate == nil ||
				cert.notAfter.After(cm.certificate.notAfter) {
				go cm.fileWrite(cert)
				cm.certificate = cert
			} else {
				cm.logger.Printf(
					"ignoring certificate which expires %s sooner\n",
					cm.certificate.notAfter.Sub(cert.notAfter))
			}
			cm.rwMutex.Unlock()
			if expire := cert.timeUntilRenewal(cm.renewBefore); expire > 0 {
				return expire
			}
		}
	}
	if err := cm.renew(); err != nil {
		cm.logger.Println(err)
		return jitteryHour()
	}
	expire := cm.certificate.timeUntilRenewal(cm.renewBefore)
	if expire < time.Hour {
		expire = jitteryHour()
	}
	return expire
}

func (cm *CertificateManager) fileLoad() error {
	if cm.certFilename == "" || cm.keyFilename == "" {
		return nil
	}
	cert, err := loadCertificate(cm.certFilename, cm.keyFilename, cm.logger)
	if err != nil {
		return err
	}
	cm.rwMutex.Lock()
	defer cm.rwMutex.Unlock()
	cm.certificate = cert
	return nil
}

func (cm *CertificateManager) fileWrite(cert *Certificate) {
	if err := cm.fileWriteError(cert); err != nil {
		cm.logger.Println(err)
	}
	select { // Non-blocking notify.
	case cm.writeNotifier <- struct{}{}:
	default:
	}
	cm.logger.Printf("wrote certificate to: %s\n", cm.certFilename)
}

func (cm *CertificateManager) fileWriteError(cert *Certificate) error {
	if cm.certFilename == "" || cm.keyFilename == "" {
		return nil
	}
	pid := os.Getpid()
	certFilename := fmt.Sprintf("%s~%d~", cm.certFilename, pid)
	keyFilename := fmt.Sprintf("%s~%d~", cm.keyFilename, pid)
	defer os.Remove(certFilename)
	defer os.Remove(keyFilename)
	err := ioutil.WriteFile(certFilename, cert.CertPemBlock, 0644)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(keyFilename, cert.KeyPemBlock, 0600)
	if err != nil {
		return err
	}
	if err := os.Rename(certFilename, cm.certFilename); err != nil {
		return err
	}
	if err := os.Rename(keyFilename, cm.keyFilename); err != nil {
		return err
	}
	return nil
}

func (cm *CertificateManager) getCertificate(hello *tls.ClientHelloInfo) (
	*tls.Certificate, error) {
	cm.rwMutex.RLock()
	defer cm.rwMutex.RUnlock()
	if cm.certificate == nil {
		return nil, errors.New("no certificate available")
	}
	return &cm.certificate.tlsCert, nil
}

func (cm *CertificateManager) makeAcmeClient(ctx context.Context) error {
	if cm.acmeClient != nil {
		return nil
	}
	key, err := makeKeyECDSA()
	if err != nil {
		return err
	}
	acmeClient := &acme.Client{
		Key:          key,
		DirectoryURL: cm.caDirectoryURL,
		UserAgent: filepath.Base(os.Args[0]) +
			" using github.com/Cloud-Foundations/golib/pkg/crypto/certmanager",
	}
	_, err = acmeClient.Register(ctx, &acme.Account{}, acme.AcceptTOS)
	if err != nil {
		return err
	}
	cm.acmeClient = acmeClient
	return nil
}

func (cm *CertificateManager) makeAcmeOrder(ctx context.Context) error {
	if err := cm.makeAcmeClient(ctx); err != nil {
		return err
	}
	if cm.acmeOrder != nil {
		if time.Now().Before(cm.acmeOrder.Expires) {
			return nil
		}
		cm.acmeOrder = nil
	}
	acmeOrder, err := cm.acmeClient.AuthorizeOrder(ctx,
		acme.DomainIDs(cm.names...))
	if err != nil {
		return err
	}
	cm.logger.Debugf(0, "ACME order will expire on: %s (in: %s)\n",
		acmeOrder.Expires.Local(),
		format.Duration(time.Until(acmeOrder.Expires)))
	cm.acmeOrder = acmeOrder
	return nil
}

// renew performs a locked ACME transaction.
func (cm *CertificateManager) renew() error {
	if err := cm.locker.Lock(); err != nil {
		return err
	}
	defer cm.locker.Unlock()
	if cm.storer != nil { // Check to see if someone else just renewed.
		if cert, _ := readCert(cm.storer); cert != nil {
			var previousNotAfter time.Time
			cm.rwMutex.RLock()
			if cm.certificate != nil {
				previousNotAfter = cm.certificate.notAfter
			}
			cm.rwMutex.RUnlock()
			if cert.notAfter.After(previousNotAfter) {
				cm.rwMutex.Lock()
				cm.certificate = cert
				cm.rwMutex.Unlock()
				go cm.fileWrite(cert)
				return nil
			}
		}
	}
	lostChannel := cm.locker.GetLostChannel()
	cert, err := cm.request(context.Background())
	if err != nil {
		return err
	}
	cm.logger.Printf("certificate issued for %s, expires on: %s (in %s)\n",
		cm.names[0], cert.notAfter.Local(),
		format.Duration(time.Until(cert.notAfter)))
	go cm.fileWrite(cert)
	cm.rwMutex.Lock()
	cm.certificate = cert
	cm.rwMutex.Unlock()
	// Write to remote storage if we kept the lock.
	select {
	case err := <-lostChannel:
		return err
	default:
	}
	if cm.storer == nil {
		return nil
	}
	return cm.storer.Write(cert)
}

// request performs an ACME request.
func (cm *CertificateManager) request(ctx context.Context) (
	*Certificate, error) {
	if err := cm.makeAcmeOrder(ctx); err != nil {
		return nil, err
	}
	for _, authoriseUrl := range cm.acmeOrder.AuthzURLs {
		if err := cm.authorise(ctx, authoriseUrl); err != nil {
			return nil, err
		}
	}
	defer cm.responder.Cleanup()
	acmeOrder, err := cm.acmeClient.WaitOrder(ctx, cm.acmeOrder.URI)
	if err != nil {
		return nil, err
	}
	cm.logger.Debugln(0, "ACME order was authorised")
	req := &x509.CertificateRequest{
		Subject:  pkix.Name{CommonName: cm.names[0]},
		DNSNames: cm.names[1:],
	}
	if cm.key == nil {
		cm.key, err = cm.keyMaker()
		if err != nil {
			return nil, err
		}
	}
	csr, err := x509.CreateCertificateRequest(rand.Reader, req, cm.key)
	if err != nil {
		return nil, err
	}
	chainDER, _, err := cm.acmeClient.CreateOrderCert(ctx,
		acmeOrder.FinalizeURL, csr, true)
	if err != nil {
		return nil, err
	}
	return makeCert(chainDER, cm.key)
}

package config

import (
	"net/http"

	"github.com/Cloud-Foundations/golib/pkg/crypto/certmanager"
	"github.com/Cloud-Foundations/golib/pkg/crypto/certmanager/dns/route53"
	cm_http "github.com/Cloud-Foundations/golib/pkg/crypto/certmanager/http"
	"github.com/Cloud-Foundations/golib/pkg/crypto/certmanager/storage/awssecretsmanager"
	"github.com/Cloud-Foundations/golib/pkg/log"
)

func newManager(certFilename, keyFilename string, httpRedirectPort uint16,
	config AcmeConfig,
	logger log.DebugLogger) (*certmanager.CertificateManager, error) {
	if config.HttpPort < 1 {
		config.HttpPort = 80
	}
	var responder certmanager.Responder
	var err error
	switch config.ChallengeType {
	case "":
	case "dns-01":
		responder, err = route53.New(config.Route53HostedZoneId,
			logger)
		if err != nil {
			return nil, err
		}
	case "http-01":
		var fallbackHandler http.Handler
		if config.HttpPort == httpRedirectPort {
			fallbackHandler = &cm_http.RedirectHandler{}
			httpRedirectPort = 0
		}
		responder, err = cm_http.NewServer(config.HttpPort,
			fallbackHandler, logger)
		if err != nil {
			return nil, err
		}
	}
	if httpRedirectPort > 0 {
		err := cm_http.CreateRedirectServer(httpRedirectPort, logger)
		if err != nil {
			return nil, err
		}
	}
	var locker certmanager.Locker
	var storer certmanager.Storer
	if config.AwsSecretId != "" {
		lockingStorer, err := awssecretsmanager.New(config.AwsSecretId, logger)
		if err != nil {
			return nil, err
		}
		locker = lockingStorer
		storer = lockingStorer
	}
	cm, err := certmanager.New(config.DomainNames, certFilename, keyFilename,
		locker, config.ChallengeType, responder, storer, 0.0, "", logger)
	if err != nil {
		return nil, err
	}
	return cm, nil
}

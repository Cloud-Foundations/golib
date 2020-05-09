/*
Package config wraps the certmanager and associated plugin packages and creates
a certificate manager based on configuration data.
*/
package config

import (
	"github.com/Cloud-Foundations/golib/pkg/crypto/certmanager"
	"github.com/Cloud-Foundations/golib/pkg/log"
)

type AcmeConfig struct {
	AwsSecretId         string   `yaml:"aws_secret_id"`
	ChallengeType       string   `yaml:"challenge_type"`
	DomainNames         []string `yaml:"domain_names"`
	HttpPort            uint16   `yaml:"http_port"`              // For http-01.
	Proxy               string   `yaml:"proxy"`                  // For http-01.
	Route53HostedZoneId string   `yaml:"route53_hosted_zone_id"` // For dns-01.
}

func New(certFilename, keyFilename string, httpRedirectPort uint16,
	config AcmeConfig,
	logger log.DebugLogger) (*certmanager.CertificateManager, error) {
	return newManager(certFilename, keyFilename, httpRedirectPort, config,
		logger)
}

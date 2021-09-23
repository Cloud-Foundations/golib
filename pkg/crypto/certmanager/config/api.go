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
	// AwsSecretId specifies the AWS secret where certificates will be stored,
	// facilitating sharing of certificates between server instances. Optional.
	AwsSecretId string `yaml:"aws_secret_id" envconfig:"ACME_AWS_SECRET_ID"`

	// ChallengeType specifies the ACME challenge type (i.e. dns-01 or http-01).
	// The default is "dns-01".
	ChallengeType string `yaml:"challenge_type" envconfig:"ACME_CHALLENGE_TYPE"`

	// DomainNames specifies the domain names (SANs) to request certificates
	// for. Required.
	DomainNames []string `yaml:"domain_names" envconfig:"ACME_DOMAIN_NAMES"`

	// HttpPort specifies the HTTP port to listen on to respond to ACME http-01
	// verification requests. The default is 80. Use this if your firewall DNATs
	// public port 80 to HttpPort internally.
	HttpPort uint16 `yaml:"http_port" envconfig:"ACME_HTTP_PORT"`

	// KeyType specifies the key type to generate, either "EC" (default) or
	// "RSA".
	KeyType string `yaml:"key_type" envconfig:"ACME_KEY_TYPE"`

	// Proxy specifies the address of a http-01 ACME proxy server. Optional.
	Proxy string `yaml:"proxy" envconfig:"ACME_PROXY"`

	// Route53HostedZoneId specifies an AWS Route53 Hosted Zone ID for the
	// dns-01 challenge. Required for the dns-01 challenge.
	Route53HostedZoneId string `yaml:"route53_hosted_zone_id" envconfig:"ACME_ROUTE53_HOSTED_ZONE_ID"`
}

func New(certFilename, keyFilename string, httpRedirectPort uint16,
	config AcmeConfig,
	logger log.DebugLogger) (*certmanager.CertificateManager, error) {
	return newManager(certFilename, keyFilename, httpRedirectPort, config,
		logger)
}

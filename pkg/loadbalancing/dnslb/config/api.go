/*
Package config wraps the dnslb and associated plugin packages and creates
a DNS load balancer based on configuration data.
*/
package config

import (
	"time"

	"github.com/Cloud-Foundations/golib/pkg/loadbalancing/dnslb"
	"github.com/Cloud-Foundations/golib/pkg/log"
)

type Config struct {
	AllRegions          bool   `yaml:"all_regions"`
	AwsProfile          string `yaml:"aws_profile"`
	dnslb.Config        `yaml:",inline"`
	Preserve            bool   `yaml:"preserve"`
	Route53HostedZoneId string `yaml:"route53_hosted_zone_id"`
}

// New creates a *dnslb.LoadBalancer using the provided configuration and
// back-end DNS provider. This will launch a goroutine to perform periodic
// health checks for the peer servers and to self register.
func New(config Config, logger log.DebugLogger) (*dnslb.LoadBalancer, error) {
	return newLoadBalancer(config, logger)
}

// Check returns true if the configuration has a single DNS back-end provider
// specified, else it returns false. An error is returned if the configuration
// is malformed (i.e. multiple DNS back-end providers specified).
func (c Config) Check() (bool, error) {
	return c.check()
}

// Block will block a server instance with the specified IP address from
// adding itself to DNS for the specified time or until a message is received on
// cancelChannel.
func Block(config Config, ip string, duration time.Duration,
	cancelChannel <-chan struct{}, logger log.DebugLogger) error {
	return block(config, ip, duration, cancelChannel, logger)
}

// RollingReplace will use the provided configuration and will roll through all
// server instances in the specified region triggering replacements by removing
// each server from DNS, destroying it and waiting for (some other mechanism) to
// create a working replacement before continuing to the next server.
func RollingReplace(config Config, region string,
	logger log.DebugLogger) error {
	return rollingReplace(config, region, logger)
}

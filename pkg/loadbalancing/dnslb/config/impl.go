package config

import (
	"errors"
	"time"

	"github.com/Cloud-Foundations/golib/pkg/loadbalancing/dnslb"
	"github.com/Cloud-Foundations/golib/pkg/loadbalancing/dnslb/route53"
	"github.com/Cloud-Foundations/golib/pkg/log"
)

type dnsConfigureFunc func(config Config,
	logger log.DebugLogger) (dnslb.RecordReadWriter, time.Duration, error)

func awsRoute53Configure(config Config,
	logger log.DebugLogger) (dnslb.RecordReadWriter, time.Duration, error) {
	rrw, err := route53.New(config.Route53HostedZoneId, logger)
	return rrw, time.Minute, err
}

func getDnsConfigureFuncs(config Config) ([]dnsConfigureFunc, error) {
	funcs := make([]dnsConfigureFunc, 0)
	if config.Route53HostedZoneId != "" {
		funcs = append(funcs, awsRoute53Configure)
	}
	if len(funcs) > 1 {
		return nil, errors.New("multiple DNS providers specified")
	}
	return funcs, nil
}

func newLoadBalancer(config Config,
	logger log.DebugLogger) (*dnslb.LoadBalancer, error) {
	funcs, err := getDnsConfigureFuncs(config)
	if err != nil {
		return nil, err
	}
	if len(funcs) < 1 {
		return nil, errors.New("no DNS zone provider specified")
	}
	recordReadWriter, defaultCheckInterval, err := funcs[0](config, logger)
	if err != nil {
		return nil, err
	}
	if defaultCheckInterval < time.Second*5 {
		defaultCheckInterval = time.Second * 5
	}
	if config.CheckInterval < 1 {
		config.CheckInterval = defaultCheckInterval
	}
	return dnslb.New(config.Config, recordReadWriter, logger)
}

func (c Config) hasDNS() (bool, error) {
	funcs, err := getDnsConfigureFuncs(c)
	if err != nil {
		return false, err
	}
	if len(funcs) < 1 {
		return false, nil
	} else {
		return true, nil
	}
}

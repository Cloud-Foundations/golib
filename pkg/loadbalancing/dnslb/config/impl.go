package config

import (
	"errors"

	"github.com/Cloud-Foundations/golib/pkg/loadbalancing/dnslb"
	"github.com/Cloud-Foundations/golib/pkg/log"
)

type dnsConfigureFunc func(config *Config, params *dnslb.Params,
	region string) error

func getDnsConfigureFuncs(config Config) ([]dnsConfigureFunc, error) {
	funcs := make([]dnsConfigureFunc, 0)
	if config.Route53HostedZoneId != "" {
		funcs = append(funcs, awsConfigure)
	}
	if len(funcs) > 1 {
		return nil, errors.New("multiple DNS providers specified")
	}
	return funcs, nil
}

func makeDnslbParams(config *Config, region string, logger log.DebugLogger) (
	*dnslb.Params, error) {
	funcs, err := getDnsConfigureFuncs(*config)
	if err != nil {
		return nil, err
	}
	if len(funcs) < 1 {
		return nil, errors.New("no DNS zone provider specified")
	}
	params := dnslb.Params{Logger: logger}
	if err := funcs[0](config, &params, region); err != nil {
		return nil, err
	}
	return &params, nil
}

func newLoadBalancer(config Config,
	logger log.DebugLogger) (*dnslb.LoadBalancer, error) {
	params, err := makeDnslbParams(&config, "", logger)
	if err != nil {
		return nil, err
	}
	return dnslb.New(config.Config, *params)
}

func (c Config) check() (bool, error) {
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

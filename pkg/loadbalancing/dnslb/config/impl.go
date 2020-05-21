package config

import (
	"errors"
	"time"

	"github.com/Cloud-Foundations/golib/pkg/awsutil/metadata"
	"github.com/Cloud-Foundations/golib/pkg/loadbalancing/dnslb"
	"github.com/Cloud-Foundations/golib/pkg/loadbalancing/dnslb/ec2"
	"github.com/Cloud-Foundations/golib/pkg/loadbalancing/dnslb/route53"
	"github.com/Cloud-Foundations/golib/pkg/log"
)

type dnsConfigureFunc func(config *Config, params *dnslb.Params) error

func awsRoute53Configure(config *Config, params *dnslb.Params) error {
	if config.CheckInterval < 1 {
		config.CheckInterval = time.Minute
	}
	var err error
	params.RecordReadWriter, err = route53.New(config.Route53HostedZoneId,
		params.Logger)
	if err != nil {
		return err
	}
	if config.AllRegions {
		if !config.Preserve {
			return errors.New("cannot destroy instances in other regions")
		}
		return nil
	}
	metadataClient, err := metadata.GetMetadataClient()
	if err != nil {
		return err
	}
	instanceHandler, err := ec2.New(metadataClient, params.Logger)
	if err != nil {
		return err
	}
	params.RegionFilter = instanceHandler
	if !config.Preserve {
		params.Destroyer = instanceHandler
	}
	return nil
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
	params := dnslb.Params{Logger: logger}
	if err := funcs[0](&config, &params); err != nil {
		return nil, err
	}
	return dnslb.New(config.Config, params)
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

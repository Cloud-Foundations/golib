package config

import (
	"github.com/Cloud-Foundations/golib/pkg/loadbalancing/dnslb"
	"github.com/Cloud-Foundations/golib/pkg/log"
)

func rollingReplace(config Config, region string,
	logger log.DebugLogger) error {
	params, err := makeDnslbParams(&config, region, logger)
	if err != nil {
		return err
	}
	return dnslb.RollingReplace(config.Config, *params, region, logger)
}

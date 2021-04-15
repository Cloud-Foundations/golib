package config

import (
	"time"

	"github.com/Cloud-Foundations/golib/pkg/loadbalancing/dnslb"
	"github.com/Cloud-Foundations/golib/pkg/log"
)

func block(config Config, ip string, duration time.Duration,
	cancelChannel <-chan struct{}, logger log.DebugLogger) error {
	params, err := makeDnslbParams(&config, "NONE", logger)
	if err != nil {
		return err
	}
	return dnslb.Block(config.Config, *params, ip, duration, cancelChannel,
		logger)
}

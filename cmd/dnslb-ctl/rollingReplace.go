package main

import (
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/golib/pkg/loadbalancing/dnslb/config"
)

func rollingReplaceSubcommand(args []string, logger log.DebugLogger) error {
	for _, region := range args {
		if err := config.RollingReplace(cfgData, region, logger); err != nil {
			return err
		}
	}
	return nil
}

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/golib/pkg/loadbalancing/dnslb/config"
)

func blockSubcommand(args []string, logger log.DebugLogger) error {
	if err := block(args[0], logger); err != nil {
		return fmt.Errorf("Error blocking IP: %s: %s", args[0], err)
	}
	return nil
}

func block(ip string, logger log.DebugLogger) error {
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)
	cancelChannel := make(chan struct{}, 1)
	go func() {
		<-signalChannel
		logger.Println("caught signal: cleaning up gracefully")
		cancelChannel <- struct{}{}
	}()
	return config.Block(cfgData, ip, *blockDuration, cancelChannel, logger)
}

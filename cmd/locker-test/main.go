package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Cloud-Foundations/Dominator/lib/flags/loadflags"
	"github.com/Cloud-Foundations/Dominator/lib/log/cmdlogger"
	"github.com/Cloud-Foundations/golib/pkg/crypto/certmanager"
	"github.com/Cloud-Foundations/golib/pkg/crypto/certmanager/storage/awssecretsmanager"
	"github.com/Cloud-Foundations/golib/pkg/log"
)

var (
	awsSecretId = flag.String("awsSecretId", "",
		"Optional AWS Secrets Manager SecretId to read/write certs to")
)

func doMain() int {
	if err := loadflags.LoadForCli("locker-test"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() != 0 {
		printUsage()
		return 3
	}
	logger := cmdlogger.New()
	if err := runLocker(logger); err != nil {
		logger.Println(err)
		return 1
	}
	return 0
}

func main() {
	os.Exit(doMain())
}

func printUsage() {
	w := flag.CommandLine.Output()
	fmt.Fprintln(w, "Usage: locker-test [flags...]")
	fmt.Fprintln(w, "Common flags:")
	flag.PrintDefaults()
}

func runLocker(logger log.DebugLogger) error {
	var locker certmanager.Locker
	if *awsSecretId != "" {
		lockingStorer, err := awssecretsmanager.New(*awsSecretId, logger)
		if err != nil {
			return err
		}
		locker = lockingStorer
	}
	if locker == nil {
		return errors.New("no locker resource specified")
	}
	if err := locker.Lock(); err != nil {
		return err
	}
	logger.Println("waiting for control-C")
	sigintChannel := make(chan os.Signal, 1)
	signal.Notify(sigintChannel, syscall.SIGINT)
	<-sigintChannel
	logger.Println()
	return locker.Unlock()
}

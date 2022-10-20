package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Cloud-Foundations/golib/pkg/awsutil/presignauth/caller"
	"github.com/Cloud-Foundations/golib/pkg/awsutil/presignauth/presigner"
	"github.com/Cloud-Foundations/golib/pkg/log/cmdlogger"
)

func printUsage() {
	w := flag.CommandLine.Output()
	fmt.Fprintln(w, "Usage: presignauth-test [flags...]")
	fmt.Fprintln(w, "Common flags:")
	flag.PrintDefaults()
}

func doMain() error {
	flag.Usage = printUsage
	flag.Parse()
	logger := cmdlogger.New(cmdlogger.GetStandardOptions())
	presignerClient, err := presigner.New(presigner.Params{
		Logger: logger,
	})
	if err != nil {
		return err
	}
	callerClient, err := caller.New(caller.Params{
		Logger: logger,
	})
	if err != nil {
		return err
	}
	logger.Printf("ARN: %s\n", presignerClient.GetCallerARN())
	presignedReq, err := presignerClient.PresignGetCallerIdentity(nil)
	if err != nil {
		return err
	}
	logger.Printf("Method: %s, URL: %s\n",
		presignedReq.Method, presignedReq.URL)
	verifiedArn, err := callerClient.GetCallerIdentity(nil, presignedReq.Method,
		presignedReq.URL)
	if err != nil {
		return err
	}
	logger.Printf("Verified ARN: %s\n", verifiedArn)
	return nil
}

func main() {
	if err := doMain(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

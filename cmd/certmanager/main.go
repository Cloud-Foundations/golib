package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Cloud-Foundations/Dominator/lib/flags/loadflags"
	"github.com/Cloud-Foundations/Dominator/lib/log/cmdlogger"
	"github.com/Cloud-Foundations/golib/pkg/crypto/certmanager"
	"github.com/Cloud-Foundations/golib/pkg/crypto/certmanager/dns/route53"
	"github.com/Cloud-Foundations/golib/pkg/crypto/certmanager/http"
	"github.com/Cloud-Foundations/golib/pkg/crypto/certmanager/storage/awssecretsmanager"
	"github.com/Cloud-Foundations/golib/pkg/log"
)

var (
	awsSecretId = flag.String("awsSecretId", "",
		"Optional AWS Secrets Manager SecretId to read/write certs to")
	cert = flag.String("cert", "",
		"file to read/write certificate from/to")
	challenge   = flag.String("challenge", "http-01", "ACME challenge type")
	dnsProvider = flag.String("dnsProvider", "route53",
		"The DNS provider to use for the dns-01 challenge")
	key        = flag.String("key", "", "file to read/write key from/to")
	portNumber = flag.Uint("portNumber", 80,
		"port number to listen on for http-01 challenge response")
	production = flag.Bool("production", false,
		"If true, use productionDirectoryURL")
	productionDirectoryURL = flag.String("productionDirectoryURL",
		certmanager.LetsEncryptProductionURL,
		"The directory endpoint for the Certificate Authority Production URL")
	route53ZoneId = flag.String("route53ZoneId", "",
		"Route 53 Hosted Zone ID for dns-01 challenge response")
	notifierCommand = flag.String("notifierCommand", "",
		"Optional command and arguments to run when the certificate is written")
	stagingDirectoryURL = flag.String("stagingDirectoryURL",
		certmanager.LetsEncryptStagingURL,
		"The directory endpoint for the Certificate Authority staging URL")
)

func getDnsResponder(logger log.DebugLogger) (certmanager.Responder, error) {
	switch *dnsProvider {
	case "manual":
		return newManualDnsResponder(), nil
	case "route53":
		return route53.New(*route53ZoneId, logger)
	default:
		return nil, fmt.Errorf("unsupported DNS provider: %s", *dnsProvider)
	}
}

func doMain() int {
	if err := loadflags.LoadForCli("certmanager"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() < 1 {
		printUsage()
		return 3
	}
	logger := cmdlogger.New()
	if err := runCertmanager(flag.Args(), logger); err != nil {
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
	fmt.Fprintln(w, "Usage: certmanager [flags...] domain...")
	fmt.Fprintln(w, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(w, "ACME challenge types:")
	fmt.Fprintln(w, "  dns-01:  respond via DNS TXT records")
	fmt.Fprintln(w, "  http-01: respond via HTTP")
	fmt.Fprintln(w, "DNS providers:")
	fmt.Fprintln(w, "  manual:  manually update DNS during ACME challenge")
	fmt.Fprintln(w, "  route53: AWS Route 53. Requires an instance role with zone write access")
}

func runCertmanager(domains []string, logger log.DebugLogger) error {
	if *cert == "" {
		return errors.New("no cert file specified")
	}
	if *key == "" {
		return errors.New("no key file specified")
	}
	var responder certmanager.Responder
	switch *challenge {
	case "dns-01":
		var err error
		responder, err = getDnsResponder(logger)
		if err != nil {
			return err
		}
	case "http-01":
		var err error
		responder, err = http.NewServer(uint16(*portNumber), nil, logger)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("challenge: %s not supported", *challenge)
	}
	directoryURL := *stagingDirectoryURL
	if *production {
		directoryURL = *productionDirectoryURL
	}
	var locker certmanager.Locker
	var storer certmanager.Storer
	if *awsSecretId != "" {
		lockingStorer, err := awssecretsmanager.New(*awsSecretId, logger)
		if err != nil {
			return err
		}
		locker = lockingStorer
		storer = lockingStorer
	}
	cm, err := certmanager.New(domains, *cert, *key, locker, *challenge,
		responder, storer, 0.0, directoryURL, logger)
	if err != nil {
		return err
	}
	logger.Println("certificate manager created")
	for range cm.GetWriteNotifier() {
		if err := runNotifier(); err != nil {
			logger.Println(err)
		}
	}
	return nil
}

func runNotifier() error {
	splitCommand := strings.Fields(*notifierCommand)
	if len(splitCommand) < 1 {
		return nil
	}
	cmd := exec.Command(splitCommand[0], splitCommand[1:]...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error running: %s: %s: %s",
			splitCommand[0], err, string(output))
	}
	return nil
}

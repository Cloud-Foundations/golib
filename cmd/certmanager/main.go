package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/Cloud-Foundations/Dominator/lib/flags/loadflags"
	"github.com/Cloud-Foundations/Dominator/lib/html"
	"github.com/Cloud-Foundations/Dominator/lib/log/serverlogger"
	"github.com/Cloud-Foundations/golib/pkg/constants"
	"github.com/Cloud-Foundations/golib/pkg/crypto/certmanager"
	"github.com/Cloud-Foundations/golib/pkg/crypto/certmanager/dns/route53"
	cm_http "github.com/Cloud-Foundations/golib/pkg/crypto/certmanager/http"
	"github.com/Cloud-Foundations/golib/pkg/crypto/certmanager/http_proxy"
	"github.com/Cloud-Foundations/golib/pkg/crypto/certmanager/storage/awssecretsmanager"
	"github.com/Cloud-Foundations/golib/pkg/log"
	"github.com/Cloud-Foundations/tricorder/go/tricorder"
)

type dashboardType struct {
	htmlWriter html.HtmlWriter
}

type htmlWriterLogger interface {
	html.HtmlWriter
	log.DebugLogger
}

var (
	adminPortNum = flag.Uint("adminPortNum", constants.CertmanagerPortNumber,
		"admin/dashboard port number to listen on")
	awsSecretId = flag.String("awsSecretId", "",
		"Optional AWS Secrets Manager SecretId to read/write certs to")
	cert = flag.String("cert", "",
		"file to read/write certificate from/to")
	challenge   = flag.String("challenge", "http-01", "ACME challenge type")
	dnsProvider = flag.String("dnsProvider", "route53",
		"The DNS provider to use for the dns-01 challenge")
	domains = flag.String("domains", "",
		"Space separated list of domains to request a certificate for")
	key     = flag.String("key", "", "file to read/write key from/to")
	portNum = flag.Uint("portNum", 80,
		"port number to listen on for http-01 challenge response")
	production = flag.Bool("production", false,
		"If true, use productionDirectoryURL")
	productionDirectoryURL = flag.String("productionDirectoryURL",
		certmanager.LetsEncryptProductionURL,
		"The directory endpoint for the Certificate Authority Production URL")
	proxyHostname = flag.String("proxyHostname", "", "hostname of acme-proxy")
	proxyPortNum  = flag.Uint("proxyPortNum", constants.AcmeProxyPortNumber,
		"port number of acme-proxy")
	redirect = flag.Bool("redirect", false,
		"If true, redirect non-ACME HTTP requests to HTTPS")
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
	flag.Usage = printUsage
	if err := loadflags.LoadForDaemon("certmanager"); err != nil {
		printUsage()
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	flag.Parse()
	tricorder.RegisterFlags()
	logger := serverlogger.New("")
	domainList := flag.Args()
	domainList = append(domainList, strings.Fields(*domains)...)
	if err := runCertmanager(domainList, logger); err != nil {
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
	fmt.Fprintln(w, "Usage: certmanager [flags...] [domain...]")
	fmt.Fprintln(w, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(w, "ACME challenge types:")
	fmt.Fprintln(w, "  dns-01:  respond via DNS TXT records")
	fmt.Fprintln(w, "  http-01: respond via HTTP")
	fmt.Fprintln(w, "DNS providers:")
	fmt.Fprintln(w, "  manual:  manually update DNS during ACME challenge")
	fmt.Fprintln(w, "  route53: AWS Route 53. Requires an instance role with zone write access")
}

func runCertmanager(domainList []string, logger htmlWriterLogger) error {
	if err := setupDashboard(logger); err != nil {
		return err
	}
	if *cert == "" {
		return errors.New("no cert file specified")
	}
	if *key == "" {
		return errors.New("no key file specified")
	}
	var responder certmanager.Responder
	var err error
	switch *challenge {
	case "dns-01":
		responder, err = getDnsResponder(logger)
	case "http-01":
		if *proxyHostname == "" {
			if *redirect {
				responder, err = cm_http.NewServer(uint16(*portNum),
					&cm_http.RedirectHandler{}, logger)
			} else {
				responder, err = cm_http.NewServer(uint16(*portNum), nil,
					logger)
			}
		} else {
			if *redirect {
				err = cm_http.CreateRedirectServer(uint16(*portNum),
					logger)
				if err != nil {
					return err
				}
			}
			responder, err = http_proxy.New(
				fmt.Sprintf("%s:%d", *proxyHostname, *proxyPortNum), logger)
		}
	default:
		return fmt.Errorf("challenge: %s not supported", *challenge)
	}
	if err != nil {
		return err
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
	cm, err := certmanager.New(domainList, *cert, *key, locker, *challenge,
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

func setupDashboard(logger htmlWriterLogger) error {
	if *adminPortNum < 1 {
		return nil
	}
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", *adminPortNum))
	if err != nil {
		return err
	}
	dashboard := &dashboardType{logger}
	html.HandleFunc("/", dashboard.statusHandler)
	go http.Serve(listener, nil)
	return nil
}

func (d *dashboardType) statusHandler(w http.ResponseWriter,
	req *http.Request) {
	if req.URL.Path != "/" {
		http.NotFound(w, req)
		return
	}
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	fmt.Fprintln(writer, "<title>certmanager status page</title>")
	fmt.Fprintln(writer, "<body>")
	fmt.Fprintln(writer, "<center>")
	fmt.Fprintln(writer, "<h1>certmanager status page</h1>")
	fmt.Fprintln(writer, "</center>")
	html.WriteHeaderWithRequest(writer, req)
	fmt.Fprintln(writer, "<h3>")
	d.htmlWriter.WriteHtml(writer)
	fmt.Fprintln(writer, "</h3>")
	fmt.Fprintln(writer, "<hr>")
	html.WriteFooter(writer)
	fmt.Fprintln(writer, "</body>")
}

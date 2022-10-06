package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/Cloud-Foundations/Dominator/lib/flags/loadflags"
	"github.com/Cloud-Foundations/Dominator/lib/html"
	"github.com/Cloud-Foundations/Dominator/lib/log/serverlogger"
	"github.com/Cloud-Foundations/golib/pkg/auth/oidc"
	"github.com/Cloud-Foundations/golib/pkg/constants"
	acmecfg "github.com/Cloud-Foundations/golib/pkg/crypto/certmanager/config"
	"github.com/Cloud-Foundations/golib/pkg/log"
	"github.com/Cloud-Foundations/tricorder/go/tricorder"
	"gopkg.in/yaml.v2"
)

var (
	configFilename = flag.String("config", "/etc/oidc-test/config.yml",
		"Configuration filename")
)

type configType struct {
	ACME                   acmecfg.AcmeConfig
	HttpRedirectPort       uint16 `yaml:"http_redirect_port"`
	ServicePort            uint16 `yaml:"service_port"`
	StatusPort             uint16 `yaml:"status_port"`
	TLSCertFilename        string `yaml:"tls_cert_filename"`
	TLSKeyFilename         string `yaml:"tls_key_filename"`
	UnencryptedServicePort uint16 `yaml:"unencrypted_service_port"`
	OpenID                 oidc.Config
}

type serverType struct {
	config *configType
	logger log.DebugLogger
}

func doMain() error {
	if err := loadflags.LoadForDaemon("oidc-test"); err != nil {
		return err
	}
	flag.Usage = printUsage
	flag.Parse()
	tricorder.RegisterFlags()
	if os.Geteuid() == 0 {
		return fmt.Errorf("Do not run the oidc-test server as root")
	}
	logger := serverlogger.New("")
	if err := startServer(logger); err != nil {
		logger.Fatalln(err)
	}
	return nil
}

func loadConfig(filename string) (*configType, error) {
	if file, err := os.Open(filename); err != nil {
		return nil, err
	} else {
		defer file.Close()
		config := configType{
			StatusPort: constants.OpenIDCTestStatusPort,
		}
		if err := yaml.NewDecoder(file).Decode(&config); err != nil {
			return nil, err
		}
		return &config, nil
	}
}

func main() {
	if err := doMain(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

func printUsage() {
	w := flag.CommandLine.Output()
	fmt.Fprintln(w, "Usage: oidc-test [flags...]")
	fmt.Fprintln(w, "Common flags:")
	flag.PrintDefaults()
}

func startServer(logger log.DebugLogger) error {
	config, err := loadConfig(*configFilename)
	if err != nil {
		return err
	}
	certmgr, err := acmecfg.New(config.TLSCertFilename, config.TLSKeyFilename,
		config.HttpRedirectPort, config.ACME, logger)
	if err != nil {
		return err
	}
	tlsConfig := tls.Config{
		GetCertificate: certmgr.GetCertificate,
		MinVersion:     tls.VersionTLS12,
	}
	server := &serverType{logger: logger}
	statusListener, err := tls.Listen("tcp",
		fmt.Sprintf(":%d", config.StatusPort),
		&tlsConfig)
	if err != nil {
		return err
	}
	html.HandleFunc("/", server.statusHandler)
	go http.Serve(statusListener, nil)
	serviceListener, err := tls.Listen("tcp",
		fmt.Sprintf(":%d", config.ServicePort),
		&tlsConfig)
	if err != nil {
		return err
	}
	serviceMux := http.NewServeMux()
	authNHandler, err := oidc.NewAuthNHandler(config.OpenID, oidc.Params{
		Handler: serviceMux,
		Logger:  logger,
	})
	if err != nil {
		return err
	}
	html.ServeMuxHandleFunc(serviceMux, "/", rootHandler)
	html.ServeMuxHandleFunc(serviceMux, "/page0", page0Handler)
	html.ServeMuxHandleFunc(serviceMux, "/page1", page1Handler)
	if config.UnencryptedServicePort > 0 {
		unencryptedListener, err := net.Listen("tcp",
			fmt.Sprintf(":%d", config.UnencryptedServicePort))
		if err != nil {
			return err
		}
		go http.Serve(unencryptedListener, authNHandler)
	}
	return http.Serve(serviceListener, authNHandler)
}

package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/Cloud-Foundations/Dominator/lib/flags/loadflags"
	"github.com/Cloud-Foundations/Dominator/lib/log/serverlogger"
	"github.com/Cloud-Foundations/golib/pkg/log"
	"github.com/Cloud-Foundations/tricorder/go/tricorder"
)

const acmePath = "/.well-known/acme-challenge"

var (
	acmePortNum = flag.Uint("acmePortNum", 80,
		"Port number to allocate and listen on for ACME http-01 challenges")
	fallbackPortNum = flag.Uint("fallbackPortNum", 0,
		"Backend port number to connect to if port 80 yields 404: Not Found")
)

type acmeProxy struct {
	logger log.DebugLogger
}

func printUsage() {
	fmt.Fprintln(os.Stderr,
		"Usage: acme-proxy [flags...]")
	fmt.Fprintln(os.Stderr, "Common flags:")
	flag.PrintDefaults()
}

func doMain() error {
	if err := loadflags.LoadForDaemon("acme-proxy"); err != nil {
		return err
	}
	flag.Usage = printUsage
	flag.Parse()
	tricorder.RegisterFlags()
	server := &acmeProxy{serverlogger.New("")}
	return http.ListenAndServe(fmt.Sprintf(":%d", *acmePortNum), server)
}

func main() {
	if err := doMain(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

func (proxy *acmeProxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	proxy.logger.Debugf(0, "source: %s, method: %s, host: %s, path: %s\n",
		req.RemoteAddr, req.Method, req.Host, req.URL.Path)
	if !strings.HasPrefix(req.URL.Path, acmePath) {
		http.Error(w, "Not an ACME challenge", http.StatusNotFound)
		return
	}
	if req.Method != "GET" {
		http.Error(w, "Use GET", http.StatusMethodNotAllowed)
		return
	}
	newUrl := "http://" + req.Host + req.URL.Path
	resp, err := http.DefaultClient.Get(newUrl)
	if err != nil {
		proxy.logger.Println(err)
		http.Error(w, "Error getting response", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound && *fallbackPortNum != 0 {
		newUrl = fmt.Sprintf("http://%s:%d%s",
			req.Host, *fallbackPortNum, req.URL.Path)
		resp, err = http.DefaultClient.Get(newUrl)
		if err != nil {
			proxy.logger.Println(err)
			http.Error(w, "Error getting response",
				http.StatusServiceUnavailable)
			return
		}
		defer resp.Body.Close()
	}
	if resp.StatusCode == http.StatusOK {
		proxy.logger.Printf("%s: OK\n", newUrl)
	} else {
		proxy.logger.Printf("%s: %s\n", newUrl, resp.Status)
	}
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		proxy.logger.Printf("%s: error copying body: %s\n", newUrl, err)
		http.Error(w, "Error reading body", http.StatusServiceUnavailable)
		return
	}
}

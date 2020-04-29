package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/Cloud-Foundations/Dominator/lib/flags/loadflags"
	"github.com/Cloud-Foundations/Dominator/lib/html"
	"github.com/Cloud-Foundations/Dominator/lib/log/serverlogger"
	"github.com/Cloud-Foundations/golib/pkg/log"
	"github.com/Cloud-Foundations/tricorder/go/tricorder"
)

const acmePath = "/.well-known/acme-challenge"

var (
	acmePortNum = flag.Uint("acmePortNum", 80,
		"Port number to allocate and listen on for ACME http-01 challenges")
	adminPortNum = flag.Uint("adminPortNum", 6941,
		"admin/dashboard port number to listen on")
	fallbackPortNum = flag.Uint("fallbackPortNum", 0,
		"Backend port number to connect to if port 80 yields 404: Not Found")
)

type acmeProxy struct {
	logger log.DebugLogger
}

type dashboardType struct {
	htmlWriter html.HtmlWriter
}

func printUsage() {
	w := flag.CommandLine.Output()
	fmt.Fprintln(w, "Usage: acme-proxy [flags...]")
	fmt.Fprintln(w, "Common flags:")
	flag.PrintDefaults()
}

func doMain() error {
	if err := loadflags.LoadForDaemon("acme-proxy"); err != nil {
		return err
	}
	flag.Usage = printUsage
	flag.Parse()
	tricorder.RegisterFlags()
	logger := serverlogger.New("")
	if err := setupDashboard(logger); err != nil {
		return err
	}
	server := &acmeProxy{logger}
	return http.ListenAndServe(fmt.Sprintf(":%d", *acmePortNum), server)
}

func main() {
	if err := doMain(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

func setupDashboard(htmlWriter html.HtmlWriter) error {
	if *adminPortNum < 1 {
		return nil
	}
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", *adminPortNum))
	if err != nil {
		return err
	}
	dashboard := &dashboardType{htmlWriter}
	html.HandleFunc("/", dashboard.statusHandler)
	go http.Serve(listener, nil)
	return nil
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

func (d *dashboardType) statusHandler(w http.ResponseWriter,
	req *http.Request) {
	if req.URL.Path != "/" {
		http.NotFound(w, req)
		return
	}
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	fmt.Fprintln(writer, "<title>acme-proxy status page</title>")
	fmt.Fprintln(writer, "<body>")
	fmt.Fprintln(writer, "<center>")
	fmt.Fprintln(writer, "<h1>acme-proxy status page</h1>")
	fmt.Fprintln(writer, "</center>")
	html.WriteHeaderWithRequest(writer, req)
	fmt.Fprintln(writer, "<h3>")
	d.htmlWriter.WriteHtml(writer)
	fmt.Fprintln(writer, "</h3>")
	fmt.Fprintln(writer, "<hr>")
	html.WriteFooter(writer)
	fmt.Fprintln(writer, "</body>")
}

package main

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/format"
	"github.com/Cloud-Foundations/golib/pkg/auth/authinfo/x509util"
)

func printUsage() {
	fmt.Fprintln(os.Stderr,
		"Usage: show-auth-cert certfile")
}

func showCert(filename string) {
	fmt.Println("Certificate:", filename+":")
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read certfile: %s\n", err)
		return
	}
	block, rest := pem.Decode(data)
	if block == nil {
		fmt.Fprintf(os.Stderr, "Failed to parse certificate PEM")
		return
	}
	if len(rest) > 0 {
		fmt.Fprintf(os.Stderr, "%d extra bytes in certfile\n", len(rest))
		return
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to parse certificate: %s\n", err)
		return
	}
	now := time.Now()
	if notYet := cert.NotBefore.Sub(now); notYet > 0 {
		fmt.Fprintf(os.Stderr, "  Will not be valid for %s\n",
			format.Duration(notYet))
	}
	if expired := now.Sub(cert.NotAfter); expired > 0 {
		fmt.Fprintf(os.Stderr, "  Expired %s ago\n", format.Duration(expired))
	}
	authInfo, err := x509util.GetAuthInfo(cert)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	authInfo.Write(os.Stdout, "  ", " ", "")
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}
	for _, filename := range os.Args[1:] {
		showCert(filename)
	}
}

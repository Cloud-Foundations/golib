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
	if authInfo.Username != "" {
		fmt.Printf("  Issued to user: %s\n", authInfo.Username)
	} else if authInfo.AwsRole != nil {
		fmt.Printf("  Issued to AWS role: %s in account: %s (ARN=%s)\n",
			authInfo.AwsRole.Name, authInfo.AwsRole.AccountId,
			authInfo.AwsRole.ARN)
	} else {
		fmt.Printf("  Issued to unknown principal: %s\n",
			cert.Subject.CommonName)
	}
	if len(authInfo.PermittedMethods) > 0 {
		fmt.Println("  Permitted methods:")
		showList(authInfo.PermittedMethods)
	} else {
		fmt.Println("  No methods are permitted")
	}
	if len(authInfo.Groups) > 0 {
		fmt.Println("  Group list:")
		showList(authInfo.Groups)
	} else {
		fmt.Println("  No group memberships")
	}
}

func showList(list []string) {
	for _, entry := range list {
		fmt.Println("   ", entry)
	}
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

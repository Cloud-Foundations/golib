/*
	Package x509util provides utility functions to process X.509 certificates.
*/
package x509util

import (
	"crypto/x509"

	"github.com/Cloud-Foundations/golib/pkg/auth/authinfo"
)

// GetAuthInfo will extract authentication information from a certificate.
func GetAuthInfo(cert *x509.Certificate) (*authinfo.AuthInfo, error) {
	return getAuthInfo(cert)
}

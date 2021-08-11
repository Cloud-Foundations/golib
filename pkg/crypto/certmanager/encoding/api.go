package encoding

import (
	"github.com/Cloud-Foundations/golib/pkg/crypto/certmanager"
)

// DecodeCert deserializes an encoded certificate into a *certmanager.Certificate.
// The encoded certificate should be the output of EncodeCert.
func DecodeCert(encodedCert string) (*certmanager.Certificate, error) {
	return decodeCert(encodedCert)
}

// EncodeCert serialized a certificiate into Base64-encoded
// DERs, and supports certificate chains.
// The output is expected be passed back to DecodeCert
func EncodeCert(cert *certmanager.Certificate) (string, error) {
	return encodeCert(cert)
}

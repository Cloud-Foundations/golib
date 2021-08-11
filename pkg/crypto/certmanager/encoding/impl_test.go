package encoding

import (
	"testing"

	"github.com/Cloud-Foundations/golib/pkg/crypto/certmanager"
)

const (
	testTypedKeyPEM = `-----BEGIN EC PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgXHeJ5aXDEz7zB7uS
k+1WujTeYzAzBgvtpOhj2mgRJdKhRANCAAQKE5puaIhI6HbXfmDpdkUimOAlVrxC
nS76isEgnr3vLchNIsWMN/94z5eMTi+bX/uQDDA5grTIETCDDBJJG/c3
-----END EC PRIVATE KEY-----
`
	testUntypedKeyPEM = `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgXHeJ5aXDEz7zB7uS
k+1WujTeYzAzBgvtpOhj2mgRJdKhRANCAAQKE5puaIhI6HbXfmDpdkUimOAlVrxC
nS76isEgnr3vLchNIsWMN/94z5eMTi+bX/uQDDA5grTIETCDDBJJG/c3
-----END PRIVATE KEY-----
`
	testCertificatePEM = `-----BEGIN CERTIFICATE-----
MIIBFDCBvAIBATAKBggqhkjOPQQDAjARMQ8wDQYDVQQDDAZUZXN0Q0EwIBcNMjAw
MzE1MDcwOTMwWhgPMjEyMDAyMjAwNzA5MzBaMBsxGTAXBgNVBAMMEFRlc3RJbnRl
cm1lZGlhdGUwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAQKE5puaIhI6HbXfmDp
dkUimOAlVrxCnS76isEgnr3vLchNIsWMN/94z5eMTi+bX/uQDDA5grTIETCDDBJJ
G/c3MAoGCCqGSM49BAMCA0cAMEQCIBYWw2ybx/ueMws2wNqEC8XtplGY8HZCA39z
S4nRrcukAiAX4PWy66NoUQGKOZsGHRKpUKNQua7KG7ysO33e+af6iw==
-----END CERTIFICATE-----

-----BEGIN CERTIFICATE-----
MIIBCzCBsgIBATAKBggqhkjOPQQDAjARMQ8wDQYDVQQDDAZUZXN0Q0EwIBcNMjAw
MzE1MDY1MzMwWhgPMjEyMDAyMjAwNjUzMzBaMBExDzANBgNVBAMMBlRlc3RDQTBZ
MBMGByqGSM49AgEGCCqGSM49AwEHA0IABHiyyDcrn5EMM58Be6viTu78UQHPWJvX
mBLDZz5i2ILLB1WF/KqeqkxlI3NhHyBbBlf0NF89ow9LNhXaHvtIkzwwCgYIKoZI
zj0EAwIDSAAwRQIhAMmltED4JLMZtowVLyFCS4ow3O6X9OKK3moaCzR6Qd6HAiAY
QjzMX8HJLQHLGYHb3FEv04EIG51pDmcPwa19BAEiLw==
-----END CERTIFICATE-----
`
)

func TestTypedKey(t *testing.T) {
	testCert := &certmanager.Certificate{
		CertPemBlock: []byte(testCertificatePEM),
		KeyPemBlock:  []byte(testTypedKeyPEM),
	}
	encodedCert, err := encodeCert(testCert)
	if err != nil {
		t.Fatal(err)
	}
	decodedCert, err := decodeCert(encodedCert)
	if err != nil {
		t.Fatal(err)
	}
	if string(decodedCert.CertPemBlock) != string(testCert.CertPemBlock) {
		t.Fatalf("decoded cert PEM: %s != test PEM: %s",
			string(decodedCert.CertPemBlock), string(testCert.CertPemBlock))
	}
	if string(decodedCert.KeyPemBlock) != string(testCert.KeyPemBlock) {
		t.Fatalf("decoded key PEM: %s != test PEM: %s",
			string(decodedCert.KeyPemBlock), string(testCert.KeyPemBlock))
	}
}

func TestUntypedKey(t *testing.T) {
	testCert := &certmanager.Certificate{
		CertPemBlock: []byte(testCertificatePEM),
		KeyPemBlock:  []byte(testUntypedKeyPEM),
	}
	encodedCert, err := encodeCert(testCert)
	if err != nil {
		t.Fatal(err)
	}
	decodedCert, err := decodeCert(encodedCert)
	if err != nil {
		t.Fatal(err)
	}
	if string(decodedCert.CertPemBlock) != string(testCert.CertPemBlock) {
		t.Fatalf("decoded cert PEM: %s != test PEM: %s",
			string(decodedCert.CertPemBlock), string(testCert.CertPemBlock))
	}
	if string(decodedCert.KeyPemBlock) != string(testCert.KeyPemBlock) {
		t.Fatalf("decoded key PEM: %s != test PEM: %s",
			string(decodedCert.KeyPemBlock), string(testCert.KeyPemBlock))
	}
}

package encoding

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	"github.com/Cloud-Foundations/golib/pkg/crypto/certmanager"
)

func decodeCert(encodedCert string) (*certmanager.Certificate, error) {
	var keyMap map[string]string
	if err := json.Unmarshal([]byte(encodedCert), &keyMap); err != nil {
		return nil, fmt.Errorf("error unmarshaling secret: %s", err)
	}
	certPEM := &bytes.Buffer{}
	for index := 0; ; index++ {
		certificateBase64 := keyMap[fmt.Sprintf("Certificate%d", index)]
		if certificateBase64 == "" {
			if index == 0 {
				return nil, errors.New("no Certificate in map")
			}
			break // We've reached the end of the certificate chain.
		}
		certDER, err := base64.StdEncoding.DecodeString(
			strings.Replace(certificateBase64, " ", "", -1))
		if err != nil {
			return nil, err
		}
		if index != 0 {
			fmt.Fprintln(certPEM)
		}
		err = pem.Encode(certPEM, &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: certDER,
		})
		if err != nil {
			return nil, err
		}
	}
	keyType := keyMap["KeyType"]
	if keyType != "" {
		keyType += " "
	}
	privateKeyBase64 := keyMap["PrivateKey"]
	if privateKeyBase64 == "" {
		return nil, errors.New("no PrivateKey in map")
	}
	privateKey, err := base64.StdEncoding.DecodeString(
		strings.Replace(privateKeyBase64, " ", "", -1))
	if err != nil {
		return nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  keyType + "PRIVATE KEY",
		Bytes: privateKey,
	})
	return &certmanager.Certificate{
		CertPemBlock: certPEM.Bytes(),
		KeyPemBlock:  keyPEM,
	}, nil
}

func encodeCert(cert *certmanager.Certificate) (string, error) {
	keyMap := make(map[string]string, 4)
	// Decode all the certificates in the chain.
	next := cert.CertPemBlock
	for index := 0; ; index++ {
		var certBlock *pem.Block
		certBlock, next = pem.Decode(next)
		if certBlock == nil {
			if index == 0 {
				return "", errors.New("unable to decode any PEM Certificate")
			}
			break // We've reached the end of the certificate chain.
		}
		if certBlock.Type != "CERTIFICATE" {
			return "", fmt.Errorf("Certificate type: %s not supported",
				certBlock.Type)
		}
		keyMap[fmt.Sprintf("Certificate%d", index)] =
			base64.StdEncoding.EncodeToString(certBlock.Bytes)
	}
	// Decode the private key.
	keyBlock, _ := pem.Decode(cert.KeyPemBlock)
	if keyBlock == nil {
		return "", errors.New("unable to decode PEM PrivateKey")
	}
	if keyBlock.Type != "PRIVATE KEY" {
		splitKeyType := strings.SplitN(keyBlock.Type, " ", 2)
		if len(splitKeyType) != 2 {
			return "", fmt.Errorf("unable to split: %s", keyBlock.Type)
		}
		if splitKeyType[1] != "PRIVATE KEY" {
			return "", fmt.Errorf("PrivateKey type: %s not supported",
				keyBlock.Type)
		}
		keyMap["KeyType"] = splitKeyType[0]
	}
	keyMap["PrivateKey"] = base64.StdEncoding.EncodeToString(keyBlock.Bytes)
	encodedCert, err := json.Marshal(keyMap)
	if err != nil {
		return "", err
	}
	return string(encodedCert), nil
}

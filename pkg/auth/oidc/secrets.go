package oidc

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"io/ioutil"
	"os"
)

func (h *authNHandler) generateSharedSecrets() error {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return err
	}
	h.sharedSecrets = []string{base64.StdEncoding.EncodeToString(buf)}
	h.params.Logger.Println("Generated shared secrets")
	return nil
}

func (h *authNHandler) loadSharedSecrets() error {
	if h.config.AwsSecretId != "" {
		return h.setupAwsSharedSecrets()
	}
	if h.config.SharedSecretFilename == "" {
		return h.generateSharedSecrets()
	}
	if file, err := os.Open(h.config.SharedSecretFilename); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := h.generateSharedSecrets(); err != nil {
			return err
		}
		h.params.Logger.Printf("Writing shared secrets to file: %s\n",
			h.config.SharedSecretFilename)
		return ioutil.WriteFile(h.config.SharedSecretFilename,
			[]byte(h.sharedSecrets[0]+"\n"), 0600)
	} else {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			h.sharedSecrets = append(h.sharedSecrets, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return err
		}
		h.params.Logger.Printf("Read shared secrets from file: %s\n",
			h.config.SharedSecretFilename)
		return nil
	}
}

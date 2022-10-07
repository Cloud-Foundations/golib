package repowatch

import (
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Cloud-Foundations/golib/pkg/log"

	"github.com/Cloud-Foundations/Dominator/lib/fsutil"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	xssh "golang.org/x/crypto/ssh"
)

func awsGetKey(secretId string, logger log.DebugLogger) (
	transport.AuthMethod, error) {
	if secretId == "" {
		if os.Getenv("SSH_AUTH_SOCK") == "" {
			logger.Debugln(0, "using default SSH configuration")
			return nil, nil // Hope for the best.
		}
		if pkc, err := ssh.NewSSHAgentAuth(ssh.DefaultUsername); err != nil {
			return nil, err
		} else {
			pkc.HostKeyCallbackHelper.HostKeyCallback =
				xssh.InsecureIgnoreHostKey()
			return pkc, nil
		}
	}
	metadataClient, err := getMetadataClient()
	if err != nil {
		return nil, err
	}
	secrets, err := getAwsSecret(metadataClient, secretId)
	if err != nil {
		return nil, err
	}
	filename, err := writeSshKey(secrets)
	if err != nil {
		return nil, err
	}
	logger.Debugf(0,
		"fetched SSH key from AWS Secrets Manager, SecretId: %s and wrote to: %s\n",
		secretId, filename)
	pubkeys, err := ssh.NewPublicKeysFromFile(ssh.DefaultUsername, filename, "")
	if err != nil {
		return nil, err
	}
	pubkeys.HostKeyCallbackHelper.HostKeyCallback = xssh.InsecureIgnoreHostKey()
	return pubkeys, nil
}

// keyMap is mutated.
func writeKeyAsPEM(writer io.Writer, keyMap map[string]string) error {
	keyType := keyMap["KeyType"]
	if keyType == "" {
		return errors.New("no KeyType in map")
	}
	delete(keyMap, "KeyType")
	privateKeyBase64 := keyMap["PrivateKey"]
	if privateKeyBase64 == "" {
		return errors.New("no PrivateKey in map")
	}
	delete(keyMap, "PrivateKey")
	privateKey, err := base64.StdEncoding.DecodeString(
		strings.Replace(privateKeyBase64, " ", "", -1))
	if err != nil {
		return err
	}
	block := &pem.Block{
		Type:    keyType + " PRIVATE KEY",
		Headers: keyMap,
		Bytes:   privateKey,
	}
	return pem.Encode(writer, block)
}

// keyMap is mutated.
func writeSshKey(keyMap map[string]string) (string, error) {
	dirname := filepath.Join(os.Getenv("HOME"), ".ssh")
	if err := os.MkdirAll(dirname, 0700); err != nil {
		return "", err
	}
	var filename string
	switch keyType := keyMap["KeyType"]; keyType {
	case "DSA":
		filename = "id_dsa"
	case "RSA":
		filename = "id_rsa"
	default:
		return "", fmt.Errorf("unsupported key type: %s", keyType)
	}
	writer, err := fsutil.CreateRenamingWriter(filepath.Join(dirname, filename),
		fsutil.PrivateFilePerms)
	if err != nil {
		return "", err
	}
	if err := writeKeyAsPEM(writer, keyMap); err != nil {
		writer.Abort()
		return "", err
	}
	return filename, writer.Close()
}

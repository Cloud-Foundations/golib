package repowatch

import (
	"context"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/Cloud-Foundations/golib/pkg/log"

	"github.com/Cloud-Foundations/Dominator/lib/fsutil"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	xssh "golang.org/x/crypto/ssh"
)

func isHttpRepo(repoURL string) bool {
	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		return false
	}
	if strings.HasPrefix(parsedURL.Scheme, "http") {
		return true
	}
	return false
}

// getAuthSSH will return with a null authentication on http repositories, and
// for ssh repostories it  tries to find an SSH authentication method.
// The ssh authenticaiton methods are AWS secrets manager, an SSH agent or
// local ssh keys.
func getAuth(ctx context.Context, repoURL string, secretsClient *secretsmanager.Client,
	secretId string, logger log.DebugLogger) (transport.AuthMethod, error) {
	if isHttpRepo(repoURL) {
		// TODO: we actually want to fetch creds from AWS or other method
		// for plain https creds.
		return nil, nil
	}
	return getAuthSSH(ctx, secretsClient, secretId, logger)
}

// getAuthSSH tries to find an SSH authentication method.
// If secretId is specified, the SSH private key will be extracted from the
// specified AWS Secrets Manager secret object, otherwise an SSH agent or local
// keys will be tried.
func getAuthSSH(ctx context.Context, secretsClient *secretsmanager.Client,
	secretId string, logger log.DebugLogger) (transport.AuthMethod, error) {
	if secretId != "" {
		return getAuthFromAWS(ctx, secretsClient, secretId, logger)
	}
	if os.Getenv("SSH_AUTH_SOCK") != "" {
		if pkc, err := ssh.NewSSHAgentAuth(ssh.DefaultUsername); err != nil {
			return nil, err
		} else {
			pkc.HostKeyCallbackHelper.HostKeyCallback =
				xssh.InsecureIgnoreHostKey()
			return pkc, nil
		}
	}
	dirname := filepath.Join(os.Getenv("HOME"), ".ssh")
	dirfile, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}
	defer dirfile.Close()
	names, err := dirfile.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	var lastError error
	for _, name := range names {
		if !strings.HasPrefix(name, "id_") {
			continue
		}
		if strings.HasSuffix(name, ".pub") {
			continue
		}
		pubkeys, err := getAuthFromFile(filepath.Join(dirname, name))
		if err == nil {
			return pubkeys, nil
		}
		lastError = err
	}
	if lastError != nil {
		return nil, lastError
	}
	return nil, fmt.Errorf("no usable SSH keys found in: %s", dirname)
}

func getAuthFromAWS(ctx context.Context, secretsClient *secretsmanager.Client,
	secretId string, logger log.DebugLogger) (
	transport.AuthMethod, error) {
	secrets, err := getAwsSecret(ctx, secretsClient, secretId)
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
	return getAuthFromFile(filename)
}

func getAuthFromFile(filename string) (transport.AuthMethod, error) {
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
	pathname := filepath.Join(dirname, filename)
	writer, err := fsutil.CreateRenamingWriter(pathname,
		fsutil.PrivateFilePerms)
	if err != nil {
		return "", err
	}
	if err := writeKeyAsPEM(writer, keyMap); err != nil {
		writer.Abort()
		return "", err
	}
	return pathname, writer.Close()
}

/*
Package awssecretsmanager implements the Locker and Storer interfaces using
AWS Secrets Manager.
*/

package awssecretsmanager

import (
	"github.com/aws/aws-sdk-go/service/secretsmanager"

	"github.com/Cloud-Foundations/golib/pkg/crypto/certmanager"
	"github.com/Cloud-Foundations/golib/pkg/log"
)

type LockingStorer struct {
	awsService *secretsmanager.SecretsManager
	logger     log.DebugLogger
	secretId   string
}

func New(secretId string, logger log.DebugLogger) (*LockingStorer, error) {
	return newLS(secretId, logger)
}

func (ls *LockingStorer) GetLostChannel() <-chan error {
	return nil
}

func (ls *LockingStorer) Lock() error {
	return ls.lock()
}

func (ls *LockingStorer) Read() (*certmanager.Certificate, error) {
	return ls.read()
}

func (ls *LockingStorer) Unlock() error {
	return ls.unlock()
}

func (ls *LockingStorer) Write(cert *certmanager.Certificate) error {
	return ls.write(cert)
}

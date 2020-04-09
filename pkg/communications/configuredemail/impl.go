package configuredemail

import (
	"fmt"
	"net/smtp"
	"os"

	"github.com/Cloud-Foundations/golib/pkg/awsutil/metadata"
	"github.com/Cloud-Foundations/golib/pkg/awsutil/secretsmgr"
	"github.com/Cloud-Foundations/golib/pkg/log"
)

type emailManager struct {
	awsSecret  *secretsmgr.CachedSecret
	logger     log.DebugLogger
	password   string
	smtpServer string
	username   string
}

func newEmailSender(config EmailConfig,
	logger log.DebugLogger) (*emailManager, error) {
	m := &emailManager{logger: logger, smtpServer: config.SmtpServer}
	if config.AwsSecretId != "" {
		metadataClient, err := metadata.GetMetadataClient()
		if err != nil {
			return nil, err
		}
		m.awsSecret, err = secretsmgr.NewCachedSecret(metadataClient,
			config.AwsSecretId, config.AwsSecretLifetime, logger)
		if err != nil {
			return nil, err
		}
	}
	if config.PasswordVariable != "" {
		m.password = os.Getenv(config.PasswordVariable)
	}
	if config.UsernameVariable != "" {
		m.username = os.Getenv(config.UsernameVariable)
	}
	return m, nil
}

func (m *emailManager) SendMail(from string, to []string, msg []byte) error {
	var username, password string
	if m.awsSecret != nil {
		var err error
		username, password, err = m.getLoginFromAws()
		if err != nil {
			return err
		}
	} else if m.username != "" && m.password != "" {
		username = username
		password = password
	}
	var auth smtp.Auth
	if username != "" && password != "" {
		auth = smtp.PlainAuth("", username, password, m.smtpServer)
	}
	return m.sendMailWithAuth(auth, from, to, msg)
}

func (m *emailManager) getLoginFromAws() (string, string, error) {
	secrets, err := m.awsSecret.GetSecret()
	if err != nil {
		return "", "", err
	}
	username, ok := secrets["Username"]
	if !ok {
		return "", "",
			fmt.Errorf("no Username in AWS Secret: %s", m.awsSecret)
	}
	password, ok := secrets["Password"]
	if !ok {
		return "", "",
			fmt.Errorf("no Password in AWS Secret: %s", m.awsSecret)
	}
	return username, password, nil
}

func (m *emailManager) sendMailWithAuth(auth smtp.Auth, from string,
	to []string, msg []byte) error {
	err := smtp.SendMail(m.smtpServer+":25", auth, from, to, msg)
	if err != nil {
		return err
	}
	m.logger.Debugf(0, "sent email from: %s to: %v\n", from, to)
	return nil
}

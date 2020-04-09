package configuredemail

import (
	"time"

	"github.com/Cloud-Foundations/golib/pkg/log"
)

type EmailConfig struct {
	AwsSecretId       string        `yaml:"aws_secret_id"`
	AwsSecretLifetime time.Duration `yaml:"aws_secret_lifetime"`
	PasswordVariable  string        `yaml:"password_variable"`
	SmtpServer        string        `yaml:"smtp_server"`
	UsernameVariable  string        `yaml:"username_variable"`
}

type EmailManager interface {
	SendMail(from string, to []string, msg []byte) error
}

func New(config EmailConfig, logger log.DebugLogger) (EmailManager, error) {
	return newEmailSender(config, logger)
}

package repowatch

import (
	"time"

	"github.com/Cloud-Foundations/golib/pkg/log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// Config specifies the configuration.
type Config struct {
	AwsSecretId              string        `yaml:"aws_secret_id"`
	Branch                   string        `yaml:"branch"`
	CheckInterval            time.Duration `yaml:"check_interval"`
	LocalRepositoryDirectory string        `yaml:"local_repository_directory"`
	RepositoryURL            string        `yaml:"repository_url"`
}

// Params specifies runtime parameters.
type Params struct {
	// Mandatory parameters.
	Logger log.DebugLogger
	// Optional parameters.
	AwsConfig       *aws.Config
	MetricDirectory string
	SecretsClient   *secretsmanager.Client
}

func Watch(config Config, params Params) (<-chan string, error) {
	return watch(config, params)
}

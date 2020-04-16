package secretsmgr

import (
	"sync"
	"time"

	"github.com/Cloud-Foundations/golib/pkg/log"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

type CachedSecret struct {
	awsService *secretsmanager.SecretsManager
	fetchTime  time.Time
	logger     log.DebugLogger
	maximumAge time.Duration
	mutex      sync.Mutex
	secretId   string
	secrets    map[string]string
}

func GetAwsSecret(metadataClient *ec2metadata.EC2Metadata,
	secretId string, logger log.DebugLogger) (map[string]string, error) {
	return getAwsSecret(metadataClient, secretId, logger)
}

func NewCachedSecret(metadataClient *ec2metadata.EC2Metadata,
	secretId string, maximumAge time.Duration,
	logger log.DebugLogger) (*CachedSecret, error) {
	return newCachedSecret(metadataClient, secretId, maximumAge, logger)
}

func (cs *CachedSecret) GetSecret() (map[string]string, error) {
	return cs.getSecret()
}

func (cs *CachedSecret) String() string {
	return cs.secretId
}

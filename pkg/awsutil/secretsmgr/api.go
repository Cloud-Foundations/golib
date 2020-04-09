package secretsmgr

import (
	"time"

	"github.com/Cloud-Foundations/golib/pkg/log"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
)

func GetAwsSecret(metadataClient *ec2metadata.EC2Metadata,
	secretId string, logger log.DebugLogger) (map[string]string, error) {
	return getAwsSecret(metadataClient, secretId, logger)
}

func GetAwsSecretWithCache(metadataClient *ec2metadata.EC2Metadata,
	secretId string, maximumAge time.Duration,
	logger log.DebugLogger) (map[string]string, error) {
	return getAwsSecretWithCache(metadataClient, secretId, maximumAge, logger)
}

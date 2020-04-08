package metadata

import (
	"errors"
	"sync"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
)

var (
	awsSecretsManagerLock                sync.Mutex
	awsSecretsManagerMetadataClient      *ec2metadata.EC2Metadata
	awsSecretsManagerMetadataClientError error
)

func getMetadataClient() (*ec2metadata.EC2Metadata, error) {
	awsSecretsManagerLock.Lock()
	defer awsSecretsManagerLock.Unlock()
	if awsSecretsManagerMetadataClient != nil {
		return awsSecretsManagerMetadataClient, nil
	}
	if awsSecretsManagerMetadataClientError != nil {
		return nil, awsSecretsManagerMetadataClientError
	}
	metadataClient := ec2metadata.New(session.New())
	if !metadataClient.Available() {
		awsSecretsManagerMetadataClientError = errors.New(
			"not running on AWS or metadata is not available")
		return nil, awsSecretsManagerMetadataClientError
	}
	awsSecretsManagerMetadataClient = metadataClient
	return awsSecretsManagerMetadataClient, nil
}

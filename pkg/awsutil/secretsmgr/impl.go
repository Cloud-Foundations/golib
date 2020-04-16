package secretsmgr

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Cloud-Foundations/golib/pkg/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

func getAwsSecret(metadataClient *ec2metadata.EC2Metadata,
	secretId string, logger log.DebugLogger) (map[string]string, error) {
	if awsService, err := makeService(metadataClient, secretId); err != nil {
		return nil, err
	} else {
		return getAwsSecretUncached(awsService, secretId, logger)
	}
}

func getAwsSecretUncached(awsService *secretsmanager.SecretsManager,
	secretId string, logger log.DebugLogger) (map[string]string, error) {
	input := secretsmanager.GetSecretValueInput{SecretId: aws.String(secretId)}
	output, err := awsService.GetSecretValue(&input)
	if err != nil {
		return nil,
			fmt.Errorf("error calling secretsmanager:GetSecretValue: %s", err)
	}
	if output.SecretString == nil {
		return nil, errors.New("no SecretString in secret")
	}
	secret := []byte(*output.SecretString)
	var secrets map[string]string
	if err := json.Unmarshal(secret, &secrets); err != nil {
		return nil, fmt.Errorf("error unmarshaling secret: %s", err)
	}
	logger.Debugf(1, "fetched AWS Secret: %s\n", secretId)
	return secrets, nil
}

func getRegion(metadataClient *ec2metadata.EC2Metadata,
	secretId string) (string, error) {
	if arn, err := arn.Parse(secretId); err == nil {
		return arn.Region, nil
	} else {
		return metadataClient.Region()
	}
}

func makeService(metadataClient *ec2metadata.EC2Metadata,
	secretId string) (*secretsmanager.SecretsManager, error) {
	region, err := getRegion(metadataClient, secretId)
	if err != nil {
		return nil, err
	}
	awsSession, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		return nil, fmt.Errorf("error creating session: %s", err)
	}
	if awsSession == nil {
		return nil, errors.New("awsSession == nil")
	}
	return secretsmanager.New(awsSession), nil
}

func newCachedSecret(metadataClient *ec2metadata.EC2Metadata,
	secretId string, maximumAge time.Duration,
	logger log.DebugLogger) (*CachedSecret, error) {
	if metadataClient == nil {
		return nil, errors.New("nil metadataClient")
	}
	if awsService, err := makeService(metadataClient, secretId); err != nil {
		return nil, err
	} else {
		logger.Debugf(1, "created cached AWS Secret: %s, lifetime: %s\n",
			secretId, maximumAge)
		return &CachedSecret{
			awsService: awsService,
			logger:     logger,
			maximumAge: maximumAge,
			secretId:   secretId,
		}, nil
	}
}

func (cs *CachedSecret) getSecret() (map[string]string, error) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	if time.Since(cs.fetchTime) < cs.maximumAge {
		cs.logger.Debugf(1, "fetched AWS Secret: %s from cache\n", cs.secretId)
		return cs.secrets, nil
	}
	secrets, err := getAwsSecretUncached(cs.awsService, cs.secretId, cs.logger)
	if err != nil {
		return nil, err
	}
	cs.fetchTime = time.Now()
	cs.secrets = secrets
	return secrets, nil
}

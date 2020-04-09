package secretsmgr

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Cloud-Foundations/golib/pkg/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

type cacheEntry struct {
	fetchTime time.Time
	secrets   map[string]string
}

var (
	cacheLock sync.Mutex
	cache     = make(map[string]cacheEntry) // Key: region/secretId.
)

func getAwsSecret(metadataClient *ec2metadata.EC2Metadata,
	secretId string, logger log.DebugLogger) (map[string]string, error) {
	region, err := getRegion(metadataClient, secretId)
	if err != nil {
		return nil, err
	}
	return getAwsSecretUncached(region, secretId, logger)
}

func getAwsSecretWithCache(metadataClient *ec2metadata.EC2Metadata,
	secretId string, maximumAge time.Duration,
	logger log.DebugLogger) (map[string]string, error) {
	region, err := getRegion(metadataClient, secretId)
	if err != nil {
		return nil, err
	}
	key := region + "/" + secretId
	cacheLock.Lock()
	defer cacheLock.Unlock()
	if entry, ok := cache[key]; ok {
		if d := time.Until(entry.fetchTime); d >= 0 && d < maximumAge {
			logger.Debugf(1, "fetched AWS Secret: %s from cache\n", secretId)
			return entry.secrets, nil
		}
		delete(cache, key)
	}
	secrets, err := getAwsSecretUncached(region, secretId, logger)
	if err != nil {
		return nil, err
	}
	cache[key] = cacheEntry{fetchTime: time.Now(), secrets: secrets}
	return secrets, nil
}

func getAwsSecretUncached(region, secretId string,
	logger log.DebugLogger) (map[string]string, error) {
	awsSession, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		return nil, fmt.Errorf("error creating session: %s", err)
	}
	if awsSession == nil {
		return nil, errors.New("awsSession == nil")
	}
	awsService := secretsmanager.New(awsSession)
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

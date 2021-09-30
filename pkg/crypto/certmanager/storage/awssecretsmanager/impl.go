package awssecretsmanager

import (
	"errors"
	"fmt"

	"github.com/Cloud-Foundations/golib/pkg/awsutil/metadata"
	"github.com/Cloud-Foundations/golib/pkg/crypto/certmanager"
	"github.com/Cloud-Foundations/golib/pkg/crypto/certmanager/encoding"
	"github.com/Cloud-Foundations/golib/pkg/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

func getAwsService(secretId string) (*secretsmanager.SecretsManager, error) {
	var region string
	if arn, err := arn.Parse(secretId); err == nil {
		region = arn.Region
	} else {
		metadataClient, err := metadata.GetMetadataClient()
		if err != nil {
			return nil, err
		}
		region, err = metadataClient.Region()
		if err != nil {
			return nil, err
		}
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

func getAwsSecret(awsService *secretsmanager.SecretsManager,
	secretId string) (string, error) {
	input := secretsmanager.GetSecretValueInput{SecretId: aws.String(secretId)}
	output, err := awsService.GetSecretValue(&input)
	if err != nil {
		return "",
			fmt.Errorf("error calling secretsmanager:GetSecretValue: %s", err)
	}
	if output.SecretString == nil {
		return "", errors.New("no SecretString in secret")
	}
	return *output.SecretString, nil
}

func newLS(secretId string, logger log.DebugLogger) (*LockingStorer, error) {
	awsService, err := getAwsService(secretId)
	if err != nil {
		return nil, err
	}
	return &LockingStorer{
		awsService: awsService,
		logger:     logger,
		secretId:   secretId,
	}, nil
}

func putAwsSecret(awsService *secretsmanager.SecretsManager,
	secretId string, secretString string) error {
	input := secretsmanager.PutSecretValueInput{
		SecretId:     aws.String(secretId),
		SecretString: aws.String(secretString),
	}
	output, err := awsService.PutSecretValue(&input)
	if err != nil {
		return fmt.Errorf("error calling secretsmanager:PutSecretValue: %s",
			err)
	}
	for _, versionStage := range output.VersionStages {
		if *versionStage == "AWSCURRENT" {
			return nil
		}
	}
	return errors.New("no AWSCURRENT version stage associated")
}

func (ls *LockingStorer) read() (*certmanager.Certificate, error) {
	secret, err := getAwsSecret(ls.awsService, ls.secretId)
	if err != nil {
		return nil, err
	}
	cert, err := encoding.DecodeCert(secret)
	if err != nil {
		return nil, err
	}
	ls.logger.Printf(
		"read certificate from AWS Secrets Manager, SecretId: %s\n",
		ls.secretId)
	return cert, nil
}

func (ls *LockingStorer) write(cert *certmanager.Certificate) error {
	secret, err := encoding.EncodeCert(cert)
	if err != nil {
		return err
	}
	if err := putAwsSecret(ls.awsService, ls.secretId, secret); err != nil {
		return err
	}
	ls.logger.Printf("wrote certificate to AWS Secrets Manager, SecretId: %s\n",
		ls.secretId)
	return nil
}

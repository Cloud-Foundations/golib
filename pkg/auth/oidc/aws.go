package oidc

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Cloud-Foundations/golib/pkg/awsutil/metadata"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

// TODO(rgooch): package up the generic code so it can be used here and in
// the pkg/crypto/certmanager/storage/awssecretsmanager package.
func getAwsService(secretId string) (*secretsmanager.SecretsManager, error) {
	metadataClient, err := metadata.GetMetadataClient()
	if err != nil {
		return nil, err
	}
	var region string
	if arn, err := arn.Parse(secretId); err == nil {
		region = arn.Region
	} else {
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
		return "", nil
	}
	return *output.SecretString, nil
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

func (h *authNHandler) setupAwsSharedSecrets() error {
	awsService, err := getAwsService(h.config.AwsSecretId)
	if err != nil {
		return err
	}
	secret, err := getAwsSecret(awsService, h.config.AwsSecretId)
	if err != nil {
		return err
	}
	if secret == "" { // Generate secret and attempt to save.
		if err := h.generateSharedSecrets(); err != nil {
			return err
		}
		err := putAwsSecret(awsService, h.config.AwsSecretId,
			fmt.Sprintf("{\"0\": \"%s\"}", h.sharedSecrets[0]))
		if err != nil {
			h.params.Logger.Println(err)
		} else {
			h.params.Logger.Printf("Wrote shared secret to AWS secret: %s\n",
				h.config.AwsSecretId)
		}
		return nil
	}
	var secrets map[string]string
	if err := json.Unmarshal([]byte(secret), &secrets); err != nil {
		return err
	}
	if len(secrets) < 1 {
		return errors.New("no entries in secrets map")
	}
	for _, value := range secrets {
		h.sharedSecrets = append(h.sharedSecrets, value)
	}
	h.params.Logger.Printf("Got shared secret from AWS secret: %s\n",
		h.config.AwsSecretId)
	return nil
}

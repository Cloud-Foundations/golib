package repowatch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

func getAwsSecret(ctx context.Context, secretsClient *secretsmanager.Client,
	secretId string) (
	map[string]string, error) {
	input := secretsmanager.GetSecretValueInput{SecretId: aws.String(secretId)}
	output, err := secretsClient.GetSecretValue(ctx, &input)
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
	return secrets, nil
}

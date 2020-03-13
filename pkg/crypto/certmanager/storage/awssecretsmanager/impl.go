package awssecretsmanager

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/Cloud-Foundations/golib/pkg/crypto/certmanager"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"

	"github.com/Cloud-Foundations/golib/pkg/log"
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

func getAwsService(secretId string) (*secretsmanager.SecretsManager, error) {
	metadataClient, err := getMetadataClient()
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
	secretId string) (map[string]string, error) {
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
	return secrets, nil
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
	secretId string, secrets map[string]string) error {
	secretString, err := json.Marshal(secrets)
	if err != nil {
		return err
	}
	input := secretsmanager.PutSecretValueInput{
		SecretId:     aws.String(secretId),
		SecretString: aws.String(string(secretString)),
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

func (ls *LockingStorer) lock() error {
	ls.logger.Printf(
		"UNIMPLEMENTED: locked AWS Secrets Manager, SecretId: %s\n",
		ls.secretId)
	return nil // HACK
}

func (ls *LockingStorer) read() (*certmanager.Certificate, error) {
	keyMap, err := getAwsSecret(ls.awsService, ls.secretId)
	if err != nil {
		return nil, err
	}
	certPEM := &bytes.Buffer{}
	for index := 0; ; index++ {
		certificateBase64 := keyMap[fmt.Sprintf("Certificate%d", index)]
		if certificateBase64 == "" {
			if index == 0 {
				return nil, errors.New("no Certificate in map")
			}
			break // We've reached the end of the certificate chain.
		}
		certDER, err := base64.StdEncoding.DecodeString(
			strings.Replace(certificateBase64, " ", "", -1))
		if err != nil {
			return nil, err
		}
		if index != 0 {
			fmt.Fprintln(certPEM)
		}
		err = pem.Encode(certPEM, &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: certDER,
		})
		if err != nil {
			return nil, err
		}
	}
	keyType := keyMap["KeyType"]
	if keyType == "" {
		return nil, errors.New("no KeyType in map")
	}
	privateKeyBase64 := keyMap["PrivateKey"]
	if privateKeyBase64 == "" {
		return nil, errors.New("no PrivateKey in map")
	}
	privateKey, err := base64.StdEncoding.DecodeString(
		strings.Replace(privateKeyBase64, " ", "", -1))
	if err != nil {
		return nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  keyType + " PRIVATE KEY",
		Bytes: privateKey,
	})
	ls.logger.Printf(
		"read certificate from AWS Secrets Manager, SecretId: %s\n",
		ls.secretId)
	return &certmanager.Certificate{
		CertPemBlock: certPEM.Bytes(),
		KeyPemBlock:  keyPEM,
	}, nil
}

func (ls *LockingStorer) unlock() error {
	ls.logger.Printf(
		"UNIMPLEMENTED: unlocked AWS Secrets Manager, SecretId: %s\n",
		ls.secretId)
	return nil // HACK
}

func (ls *LockingStorer) write(cert *certmanager.Certificate) error {
	keyMap := make(map[string]string, 4)
	// Decode all the certificates in the chain.
	next := cert.CertPemBlock
	for index := 0; ; index++ {
		var certBlock *pem.Block
		certBlock, next = pem.Decode(next)
		if certBlock == nil {
			if index == 0 {
				return errors.New("unable to decode any PEM Certificate")
			}
			break // We've reached the end of the certificate chain.
		}
		if certBlock.Type != "CERTIFICATE" {
			return fmt.Errorf("Certificate type: %s not supported",
				certBlock.Type)
		}
		keyMap[fmt.Sprintf("Certificate%d", index)] =
			base64.StdEncoding.EncodeToString(certBlock.Bytes)
	}
	// Decode the private key.
	keyBlock, _ := pem.Decode(cert.KeyPemBlock)
	if keyBlock == nil {
		return errors.New("unable to decode PEM PrivateKey")
	}
	splitKeyType := strings.SplitN(keyBlock.Type, " ", 2)
	if len(splitKeyType) != 2 {
		return fmt.Errorf("unable to split: %s", keyBlock.Type)
	}
	if splitKeyType[1] != "PRIVATE KEY" {
		return fmt.Errorf("PrivateKey type: %s not supported", keyBlock.Type)
	}
	keyMap["KeyType"] = splitKeyType[0]
	keyMap["PrivateKey"] = base64.StdEncoding.EncodeToString(keyBlock.Bytes)
	if err := putAwsSecret(ls.awsService, ls.secretId, keyMap); err != nil {
		return err
	}
	ls.logger.Printf("wrote certificate to AWS Secrets Manager, SecretId: %s\n",
		ls.secretId)
	return nil
}

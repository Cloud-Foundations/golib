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

func decodeCert(encodedCert string) (*certmanager.Certificate, error) {
	var keyMap map[string]string
	if err := json.Unmarshal([]byte(encodedCert), &keyMap); err != nil {
		return nil, fmt.Errorf("error unmarshaling secret: %s", err)
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
	if keyType != "" {
		keyType += " "
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
		Type:  keyType + "PRIVATE KEY",
		Bytes: privateKey,
	})
	return &certmanager.Certificate{
		CertPemBlock: certPEM.Bytes(),
		KeyPemBlock:  keyPEM,
	}, nil
}

func encodeCert(cert *certmanager.Certificate) (string, error) {
	keyMap := make(map[string]string, 4)
	// Decode all the certificates in the chain.
	next := cert.CertPemBlock
	for index := 0; ; index++ {
		var certBlock *pem.Block
		certBlock, next = pem.Decode(next)
		if certBlock == nil {
			if index == 0 {
				return "", errors.New("unable to decode any PEM Certificate")
			}
			break // We've reached the end of the certificate chain.
		}
		if certBlock.Type != "CERTIFICATE" {
			return "", fmt.Errorf("Certificate type: %s not supported",
				certBlock.Type)
		}
		keyMap[fmt.Sprintf("Certificate%d", index)] =
			base64.StdEncoding.EncodeToString(certBlock.Bytes)
	}
	// Decode the private key.
	keyBlock, _ := pem.Decode(cert.KeyPemBlock)
	if keyBlock == nil {
		return "", errors.New("unable to decode PEM PrivateKey")
	}
	if keyBlock.Type != "PRIVATE KEY" {
		splitKeyType := strings.SplitN(keyBlock.Type, " ", 2)
		if len(splitKeyType) != 2 {
			return "", fmt.Errorf("unable to split: %s", keyBlock.Type)
		}
		if splitKeyType[1] != "PRIVATE KEY" {
			return "", fmt.Errorf("PrivateKey type: %s not supported",
				keyBlock.Type)
		}
		keyMap["KeyType"] = splitKeyType[0]
	}
	keyMap["PrivateKey"] = base64.StdEncoding.EncodeToString(keyBlock.Bytes)
	encodedCert, err := json.Marshal(keyMap)
	if err != nil {
		return "", err
	}
	return string(encodedCert), nil
}

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
	cert, err := decodeCert(secret)
	if err != nil {
		return nil, err
	}
	ls.logger.Printf(
		"read certificate from AWS Secrets Manager, SecretId: %s\n",
		ls.secretId)
	return cert, nil
}

func (ls *LockingStorer) write(cert *certmanager.Certificate) error {
	secret, err := encodeCert(cert)
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

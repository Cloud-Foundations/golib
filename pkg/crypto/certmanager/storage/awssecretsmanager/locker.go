package awssecretsmanager

import (
	"crypto/rand"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

func sleep() {
	randByte := make([]byte, 1)
	rand.Read(randByte)
	time.Sleep(time.Second*15 + time.Millisecond*10*time.Duration(randByte[0]))

}

// This must be called with the lock held.
func (ls *LockingStorer) breakLock() error {
	getInput := secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(ls.secretId),
		VersionStage: aws.String("LOCK"),
	}
	getOutput, err := ls.awsService.GetSecretValue(&getInput)
	if err != nil {
		return fmt.Errorf("error calling secretsmanager:GetSecretValue: %s",
			err)
	}
	// Remove LOCK label unless it's not time to break yet.
	if getOutput.SecretString == nil {
		ls.logger.Printf("no SecretString in SecretId: %s label: LOCK\n",
			ls.secretId)
	} else {
		expirationEpoch, err := strconv.ParseInt(*getOutput.SecretString, 10,
			64)
		if err != nil {
			ls.logger.Printf("error parsing epoch: \"%s\": %s\n",
				*getOutput.SecretString, err)
		} else if expirationEpoch > time.Now().Unix() {
			return nil // Not yet time to break.
		}
	}
	updateInput := secretsmanager.UpdateSecretVersionStageInput{
		RemoveFromVersionId: getOutput.VersionId,
		SecretId:            aws.String(ls.secretId),
		VersionStage:        aws.String("LOCK"),
	}
	_, err = ls.awsService.UpdateSecretVersionStage(&updateInput)
	if err != nil {
		return fmt.Errorf("unable to remove LOCK label for SecretId: %s\n",
			ls.secretId)
	}
	ls.logger.Printf("broke lock for SecretId: %s, version: %s\n",
		ls.secretId, *getOutput.VersionId)
	return nil
}

func (ls *LockingStorer) lock() error {
	ls.mutex.Lock()
	defer ls.mutex.Unlock()
	if ls.lockVersion != nil {
		return fmt.Errorf("already locked SecretId: %s", ls.secretId)
	}
	expirationTime := time.Now().Add(time.Minute * 15)
	putInput := secretsmanager.PutSecretValueInput{
		SecretId:      aws.String(ls.secretId),
		SecretString:  aws.String(fmt.Sprintf("%d", expirationTime.Unix())),
		VersionStages: aws.StringSlice([]string{"DUMMY"}),
	}
	putOutput, err := ls.awsService.PutSecretValue(&putInput)
	if err != nil {
		return fmt.Errorf("error calling secretsmanager:PutSecretValue: %s",
			err)
	}
	updateInput := secretsmanager.UpdateSecretVersionStageInput{
		MoveToVersionId: putOutput.VersionId,
		SecretId:        aws.String(ls.secretId),
		VersionStage:    aws.String("LOCK"),
	}
	for {
		_, err := ls.awsService.UpdateSecretVersionStage(&updateInput)
		if err == nil {
			ls.lockVersion = putOutput.VersionId
			break
		}
		ls.logger.Debugf(0,
			"error grabbing lock for SecretId: %s, waiting: %s\n",
			ls.secretId, err)
		sleep()
		if err := ls.breakLock(); err != nil {
			ls.logger.Println(err)
		}
	}
	updateInput = secretsmanager.UpdateSecretVersionStageInput{
		RemoveFromVersionId: putOutput.VersionId,
		SecretId:            aws.String(ls.secretId),
		VersionStage:        aws.String("DUMMY"),
	}
	_, err = ls.awsService.UpdateSecretVersionStage(&updateInput)
	if err != nil {
		ls.logger.Printf("unable to remove DUMMY label for SecretId: %s\n",
			ls.secretId)
	}
	ls.logger.Printf("locked AWS Secrets Manager, SecretId: %s\n", ls.secretId)
	return nil
}

func (ls *LockingStorer) unlock() error {
	ls.mutex.Lock()
	defer ls.mutex.Unlock()
	if ls.lockVersion == nil {
		return fmt.Errorf("already unlocked SecretId: %s", ls.secretId)
	}
	updateInput := secretsmanager.UpdateSecretVersionStageInput{
		RemoveFromVersionId: ls.lockVersion,
		SecretId:            aws.String(ls.secretId),
		VersionStage:        aws.String("LOCK"),
	}
	_, err := ls.awsService.UpdateSecretVersionStage(&updateInput)
	if err != nil {
		return fmt.Errorf("unable to remove LOCK label for SecretId: %s",
			ls.secretId)
	}
	ls.lockVersion = nil
	ls.logger.Printf("unlocked AWS Secrets Manager, SecretId: %s\n",
		ls.secretId)
	return nil
}

package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/Cloud-Foundations/golib/pkg/awsutil/metadata"
	"github.com/Cloud-Foundations/golib/pkg/dns/route53"
	"github.com/Cloud-Foundations/golib/pkg/loadbalancing/dnslb"
	"github.com/Cloud-Foundations/golib/pkg/loadbalancing/dnslb/ec2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
)

func awsConfigure(config *Config, params *dnslb.Params, region string) error {
	if config.CheckInterval < 1 {
		config.CheckInterval = time.Minute
	}
	awsSession, err := awsCreateSession(config)
	if err != nil {
		return err
	}
	if err := awsEC2Configure(awsSession, config, params, region); err != nil {
		return err
	}
	if err := awsRoute53Configure(awsSession, config, params); err != nil {
		return err
	}
	return nil
}

func awsCreateSession(config *Config) (*session.Session, error) {
	var awsSession *session.Session
	var err error
	if config.AwsProfile == "" {
		awsSession, err = session.NewSession(&aws.Config{})
	} else {
		awsSession, err = session.NewSessionWithOptions(session.Options{
			Profile: config.AwsProfile,
		})
	}
	if err != nil {
		return nil, fmt.Errorf("error creating session: %s", err)
	}
	if awsSession == nil {
		return nil, errors.New("awsSession == nil")
	}
	if config.AwsAssumeRoleArn == "" {
		return awsSession, nil
	}
	creds := stscreds.NewCredentials(awsSession, config.AwsAssumeRoleArn)
	assumedSession, err := session.NewSession(&aws.Config{Credentials: creds})
	if err != nil {
		return nil, fmt.Errorf("error creating assumed role session: %s", err)
	}
	if assumedSession == nil {
		return nil, errors.New("assumedSession == nil")
	}
	return assumedSession, nil
}

func awsEC2Configure(awsSession *session.Session, config *Config,
	params *dnslb.Params, region string) error {
	if config.AllRegions {
		if !config.Preserve {
			return errors.New("cannot destroy instances in other regions")
		}
		return nil
	}
	if region == "" {
		metadataClient, err := metadata.GetMetadataClient()
		if err != nil {
			return err
		}
		region, err = metadataClient.Region()
		if err != nil {
			return err
		}
	}
	instanceHandler, err := ec2.New(awsSession, region, params.Logger)
	if err != nil {
		return err
	}
	params.RegionFilter = instanceHandler
	if !config.Preserve {
		params.Destroyer = instanceHandler
	}
	return nil
}

func awsRoute53Configure(awsSession *session.Session, config *Config,
	params *dnslb.Params) error {
	var err error
	params.RecordReadWriter, err = route53.New(awsSession,
		config.Route53HostedZoneId, params.Logger)
	if err != nil {
		return err
	}
	return nil
}

package presigner

import (
	"context"
	"time"

	"github.com/Cloud-Foundations/golib/pkg/awsutil/presignauth"
	"github.com/Cloud-Foundations/golib/pkg/log/nulllogger"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

const (
	presignedUrlLifetime = 15*time.Minute - 7*time.Second
)

func newPresigner(params Params) (*presignerT, error) {
	ctx := context.Background()
	if params.Logger == nil {
		params.Logger = nulllogger.New()
	}
	if params.StsClient == nil {
		if params.AwsConfig == nil {
			awsConfig, err := config.LoadDefaultConfig(ctx,
				config.WithEC2IMDSRegion())
			if err != nil {
				return nil, err
			}
			params.AwsConfig = &awsConfig
		}
		params.StsClient = sts.NewFromConfig(*params.AwsConfig)
	}
	if params.StsPresignClient == nil {
		params.StsPresignClient = sts.NewPresignClient(params.StsClient)
	}
	idOutput, err := params.StsClient.GetCallerIdentity(ctx,
		&sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, err
	}
	parsedArn, err := arn.Parse(*idOutput.Arn)
	if err != nil {
		return nil, err
	}
	normalisedArn, err := presignauth.NormaliseARN(parsedArn)
	if err != nil {
		return nil, err
	}
	callerArn := normalisedArn.String()
	params.Logger.Debugf(0,
		"Account: %s, RawARN: %s, NormalisedARN: %s, UserId: %s\n",
		*idOutput.Account, *idOutput.Arn, callerArn, *idOutput.UserId)
	presigner := &presignerT{
		params:    params,
		callerArn: normalisedArn,
	}
	if params.RefreshPolicy == RefreshAutomatically {
		go presigner.refreshLoop(ctx)
	}
	return presigner, nil
}

func (p *presignerT) presignGetCallerIdentity(ctx context.Context) (
	*v4.PresignedHTTPRequest, error) {
	if ctx == nil {
		ctx = context.TODO()
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.presignedRequest != nil {
		if time.Until(p.presignedExpiration) > 0 {
			return p.presignedRequest, nil
		}
		p.presignedRequest = nil
	}
	presignedReq, err := p.params.StsPresignClient.PresignGetCallerIdentity(ctx,
		&sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, err
	}
	p.presignedExpiration = time.Now().Add(presignedUrlLifetime)
	p.presignedRequest = presignedReq
	p.params.Logger.Debugf(2, "presigned URL: %v\n", presignedReq.URL)
	return presignedReq, nil
}

func (p *presignerT) refreshLoop(ctx context.Context) {
	for ; ; time.Sleep(presignedUrlLifetime) {
		if _, err := p.presignGetCallerIdentity(ctx); err != nil {
			p.params.Logger.Println(err)
		}
	}
}

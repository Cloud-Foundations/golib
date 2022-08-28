package presigner

import (
	"context"
	"sync"
	"time"

	"github.com/Cloud-Foundations/golib/pkg/log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

const (
	RefreshOnDemand = iota
	RefreshAutomatically
)

type Params struct {
	// Optional parameters.
	AwsConfig        *aws.Config
	Logger           log.DebugLogger
	RefreshPolicy    uint // Default is RefreshOnDemand.
	StsClient        *sts.Client
	StsPresignClient *sts.PresignClient
}

type Presigner interface {
	GetCallerARN() arn.ARN
	PresignGetCallerIdentity(ctx context.Context) (
		*v4.PresignedHTTPRequest, error)
}

type presignerT struct {
	callerArn           arn.ARN
	params              Params
	mutex               sync.Mutex // Protect everything below.
	presignedExpiration time.Time
	presignedRequest    *v4.PresignedHTTPRequest
}

// Interface checks.
var _ Presigner = (*presignerT)(nil)

// New will create a presigner client which caches presigned URLs until they
// expire (~15 minutes).
func New(params Params) (*presignerT, error) {
	return newPresigner(params)
}

// GetCallerARN will get the normalised ARN of the caller. The ARN will have the
// form: arn:aws:iam::$AccountId:role/$RoleName
func (p *presignerT) GetCallerARN() arn.ARN { return p.callerArn }

// PresignGetCallerIdentity will generate a presigned URL (token) which may be
// used to verify the AWS IAM identity of the token bearer.
func (p *presignerT) PresignGetCallerIdentity(ctx context.Context) (
	*v4.PresignedHTTPRequest, error) {
	return p.presignGetCallerIdentity(ctx)
}

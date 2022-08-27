package presignauth

import (
	"github.com/aws/aws-sdk-go-v2/aws/arn"
)

// NormaliseARN will normalise an AWS IAM ARN (i.e. an ARN returned from
// sts:GetCallerIdentity), returning the actual role ARN, rather than an ARN
// showing how the credentials were obtained (such as by assuming the role).
// This mirrors the way AWS policy documents are written. The ARN will have the
// form: arn:aws:iam::$AccountId:role/$RoleName
func NormaliseARN(input arn.ARN) (arn.ARN, error) {
	return normaliseARN(input)
}

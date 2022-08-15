package presignauth

import (
	"github.com/aws/aws-sdk-go-v2/aws/arn"
)

func NormaliseARN(input arn.ARN) (arn.ARN, error) {
	return normaliseARN(input)
}

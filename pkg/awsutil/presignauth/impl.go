package presignauth

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
)

func normaliseARN(input arn.ARN) (arn.ARN, error) {
	switch input.Service {
	case "iam", "sts":
	default:
		return arn.ARN{}, fmt.Errorf("unsupported service: %s", input.Service)
	}
	splitResource := strings.Split(input.Resource, "/")
	if len(splitResource) < 2 || splitResource[0] != "assumed-role" {
		return arn.ARN{}, fmt.Errorf("invalid resource: %s", input.Resource)
	}
	return arn.ARN{
		Partition: input.Partition,
		Service:   "iam",
		AccountID: input.AccountID,
		Resource:  "role/" + splitResource[1],
	}, nil
}

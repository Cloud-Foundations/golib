package metadata

import (
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
)

func GetMetadataClient() (*ec2metadata.EC2Metadata, error) {
	return getMetadataClient()
}

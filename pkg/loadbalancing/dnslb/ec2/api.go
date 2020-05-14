package ec2

import (
	"sync"

	"github.com/Cloud-Foundations/golib/pkg/log"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type InstanceHandler struct {
	awsService   *ec2.EC2
	logger       log.DebugLogger
	mutex        sync.Mutex         // Protect everything below.
	ipToInstance map[string]*string // Key: IP, value instance ID.
}

func New(metadataClient *ec2metadata.EC2Metadata,
	logger log.DebugLogger) (*InstanceHandler, error) {
	return newInstanceHandler(metadataClient, logger)
}

func (h *InstanceHandler) Destroy(ips map[string]struct{}) error {
	return h.destroy(ips)
}

func (h *InstanceHandler) Filter(ips map[string]struct{}) (
	map[string]struct{}, error) {
	return h.filter(ips)
}

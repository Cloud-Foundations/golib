package ec2

import (
	"fmt"

	"github.com/Cloud-Foundations/golib/pkg/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type ipMap map[string]struct{}

func newInstanceHandler(awsSession *session.Session, region string,
	logger log.DebugLogger) (*InstanceHandler, error) {
	awsSession = awsSession.Copy(&aws.Config{Region: aws.String(region)})
	return &InstanceHandler{
		awsService:   ec2.New(awsSession),
		logger:       logger,
		ipToInstance: make(map[string]*string),
	}, nil
}

func (h *InstanceHandler) destroy(ips ipMap) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	instances, err := h.getInstanceIDs(ips)
	if err != nil {
		return err
	}
	if len(instances) < 1 {
		return nil
	}
	_, err = h.awsService.TerminateInstances(
		&ec2.TerminateInstancesInput{InstanceIds: instances})
	if err != nil {
		return fmt.Errorf("ec2:TerminateInstances: %s", err)
	}
	for ip := range ips {
		if h.ipToInstance[ip] != nil {
			delete(h.ipToInstance, ip)
		}
	}
	return nil
}

func (h *InstanceHandler) filter(ips ipMap) (ipMap, error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if _, err := h.getInstanceIDs(ips); err != nil {
		return nil, err
	}
	filtered := make(ipMap, len(ips))
	for ip := range ips {
		if h.ipToInstance[ip] != nil {
			filtered[ip] = struct{}{}
		}
	}
	return filtered, nil
}

// Must be called with lock held. Returns instances in the same region.
func (h *InstanceHandler) getInstanceIDs(ips ipMap) ([]*string, error) {
	if ids := h.getInstanceIDsCached(ips); ids != nil {
		return ids, nil
	}
	awsIPs := make([]*string, 0, len(ips))
	for ip := range ips {
		awsIPs = append(awsIPs, aws.String(ip))
	}
	output, err := h.awsService.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{{
			Name:   aws.String("private-ip-address"),
			Values: awsIPs,
		}},
	})
	if err != nil {
		return nil, fmt.Errorf("ec2:DescribeInstances: %s", err)
	}
	ipToInstance := make(map[string]*string, len(ips))
	for ip := range ips {
		ipToInstance[ip] = nil
	}
	for _, reservation := range output.Reservations {
		for _, instance := range reservation.Instances {
			ipToInstance[*instance.PrivateIpAddress] = instance.InstanceId
		}
	}
	h.ipToInstance = ipToInstance
	return h.getInstanceIDsCached(ips), nil
}

// Must be called with lock held. Returns nil if the cache is incomplete.
func (h *InstanceHandler) getInstanceIDsCached(ips ipMap) []*string {
	instanceIDs := make([]*string, 0, len(ips))
	for ip := range ips {
		if instanceId, ok := h.ipToInstance[ip]; !ok {
			return nil
		} else if instanceId != nil {
			instanceIDs = append(instanceIDs, instanceId)
		}
	}
	return instanceIDs
}

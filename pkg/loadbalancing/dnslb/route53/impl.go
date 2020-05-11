package route53

import (
	"errors"
	"time"

	"github.com/Cloud-Foundations/golib/pkg/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
)

func newRecordReadWriter(hostedZoneId string,
	logger log.DebugLogger) (*RecordReadWriter, error) {
	if hostedZoneId == "" {
		return nil, errors.New("no hosted zone ID specified")
	}
	awsSession, err := session.NewSession(&aws.Config{})
	if err != nil {
		return nil, err
	}
	if awsSession == nil {
		return nil, errors.New("awsSession == nil")
	}
	return &RecordReadWriter{
		awsService:   route53.New(awsSession),
		hostedZoneId: aws.String(hostedZoneId),
		logger:       logger,
	}, nil
}

func (rrw *RecordReadWriter) readRecord(fqdn string) ([]string, error) {
	if fqdn[len(fqdn)-1] != '.' {
		fqdn += "."
	}
	input := &route53.ListResourceRecordSetsInput{
		HostedZoneId:    rrw.hostedZoneId,
		StartRecordName: aws.String(fqdn),
		StartRecordType: aws.String("A"),
	}
	output, err := rrw.awsService.ListResourceRecordSets(input)
	if err != nil {
		return nil, err
	}
	var ips []string
	for _, recordSet := range output.ResourceRecordSets {
		name := *recordSet.Name
		if name[0] == '"' && name[len(name)-1] == '"' {
			name = name[1 : len(name)-1]
		}
		if name != fqdn || *recordSet.Type != "A" {
			continue
		}
		for _, record := range recordSet.ResourceRecords {
			ips = append(ips, *record.Value)
		}
	}
	return ips, nil
}

func (rrw *RecordReadWriter) writeRecord(fqdn string, ips []string,
	ttl time.Duration) error {
	var resourceRecords []*route53.ResourceRecord
	for _, ip := range ips {
		resourceRecords = append(resourceRecords,
			&route53.ResourceRecord{Value: aws.String(ip)})
	}
	input := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{{
				Action: aws.String("UPSERT"),
				ResourceRecordSet: &route53.ResourceRecordSet{
					Name:            aws.String(fqdn),
					ResourceRecords: resourceRecords,
					TTL:             aws.Int64(int64(ttl.Seconds())),
					Type:            aws.String("A"),
				}},
			},
		},
		HostedZoneId: rrw.hostedZoneId,
	}
	_, err := rrw.awsService.ChangeResourceRecordSets(input)
	return err
}

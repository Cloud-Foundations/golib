package route53

import (
	"errors"
	"time"

	"github.com/Cloud-Foundations/golib/pkg/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
)

func newRecordReadWriter(awsSession *session.Session, hostedZoneId string,
	logger log.DebugLogger) (*RecordReadWriter, error) {
	if hostedZoneId == "" {
		return nil, errors.New("no hosted zone ID specified")
	}
	return &RecordReadWriter{
		awsService:   route53.New(awsSession),
		hostedZoneId: aws.String(hostedZoneId),
		logger:       logger,
	}, nil
}

// Insert double quotes if missing.
func insertQuotes(value string) string {
	if value[0] == '"' && value[len(value)-1] == '"' {
		return value
	}
	return "\"" + value + "\""
}

// Strip double quotes if present.
func stripQuotes(value string) string {
	if value[0] == '"' && value[len(value)-1] == '"' {
		value = value[1 : len(value)-1]
	}
	return value
}

func (rrw *RecordReadWriter) readRecords(fqdn string, recType string) (
	[]string, time.Duration, error) {
	if fqdn[len(fqdn)-1] != '.' {
		fqdn += "."
	}
	input := &route53.ListResourceRecordSetsInput{
		HostedZoneId:    rrw.hostedZoneId,
		StartRecordName: aws.String(fqdn),
		StartRecordType: aws.String(recType),
	}
	output, err := rrw.awsService.ListResourceRecordSets(input)
	if err != nil {
		return nil, 0, err
	}
	var ttl time.Duration
	var records []string
	for _, recordSet := range output.ResourceRecordSets {
		name := stripQuotes(*recordSet.Name)
		if name != fqdn || *recordSet.Type != recType {
			continue
		}
		if _ttl := time.Duration(*recordSet.TTL) * time.Second; _ttl > ttl {
			ttl = _ttl
		}
		for _, record := range recordSet.ResourceRecords {
			records = append(records, stripQuotes(*record.Value))
		}
	}
	return records, ttl, nil
}

func (rrw *RecordReadWriter) writeRecords(fqdn, recType string,
	records []string, ttl time.Duration) error {
	if fqdn[len(fqdn)-1] != '.' {
		fqdn += "."
	}
	var resourceRecords []*route53.ResourceRecord
	for _, record := range records {
		if recType == "TXT" {
			record = insertQuotes(record)
		}
		resourceRecords = append(resourceRecords,
			&route53.ResourceRecord{Value: aws.String(record)})
	}
	input := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{{
				Action: aws.String("UPSERT"),
				ResourceRecordSet: &route53.ResourceRecordSet{
					Name:            aws.String(fqdn),
					ResourceRecords: resourceRecords,
					TTL:             aws.Int64(int64(ttl.Seconds())),
					Type:            aws.String(recType),
				}},
			},
		},
		HostedZoneId: rrw.hostedZoneId,
	}
	_, err := rrw.awsService.ChangeResourceRecordSets(input)
	return err
}

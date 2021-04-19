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

func waitForChange(awsService *route53.Route53, id *string,
	logger log.DebugLogger) error {
	timer := time.NewTimer(time.Minute * 2)
	errorChannel := make(chan error, 1)
	go func() {
		errorChannel <- awsService.WaitUntilResourceRecordSetsChanged(
			&route53.GetChangeInput{Id: id})
	}()
	select {
	case <-timer.C:
		output, err := awsService.GetChange(&route53.GetChangeInput{Id: id})
		if err != nil {
			logger.Printf("timed out waiting for change: %s, hoping for the best, error from GetChange(): %s\n",
				*id, err)
			return nil
		}
		logger.Printf(
			"timed out waiting for change: %s, hoping for the best, status: %s\n",
			id, *output.ChangeInfo.Status)
		return nil
	case err := <-errorChannel:
		return err
	}
}

func (rrw *RecordReadWriter) deleteRecords(fqdn, recType string) error {
	if fqdn[len(fqdn)-1] != '.' {
		fqdn += "."
	}
	output, err := rrw.awsService.ListResourceRecordSets(
		&route53.ListResourceRecordSetsInput{
			HostedZoneId:    rrw.hostedZoneId,
			StartRecordName: aws.String(fqdn),
			StartRecordType: aws.String(recType),
		})
	if err != nil {
		return err
	}
	var changes []*route53.Change
	for _, recordSet := range output.ResourceRecordSets {
		name := stripQuotes(*recordSet.Name)
		if name != fqdn || *recordSet.Type != recType {
			continue
		}
		changes = append(changes, &route53.Change{
			Action:            aws.String("DELETE"),
			ResourceRecordSet: recordSet})
	}
	input := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch:  &route53.ChangeBatch{Changes: changes},
		HostedZoneId: rrw.hostedZoneId,
	}
	_, err = rrw.awsService.ChangeResourceRecordSets(input)
	return err
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
	records []string, ttl time.Duration, wait bool) error {
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
	output, err := rrw.awsService.ChangeResourceRecordSets(input)
	if err != nil {
		return err
	}
	if wait {
		rrw.logger.Debugf(1, "waiting for change: %s to complete\n",
			*output.ChangeInfo.Id)
		err = waitForChange(rrw.awsService, output.ChangeInfo.Id, rrw.logger)
		if err != nil {
			return err
		}
		rrw.logger.Debugf(1, "change: %s completed\n", *output.ChangeInfo.Id)
	}
	return nil
}

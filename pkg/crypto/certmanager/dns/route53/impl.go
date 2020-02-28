package route53

import (
	"errors"
	"time"

	"github.com/Cloud-Foundations/golib/pkg/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
)

// const defaultRegion = "us-west-2"

func makeTXT(action, fqdn, txtValue string) *route53.Change {
	return &route53.Change{
		Action: aws.String(action),
		ResourceRecordSet: &route53.ResourceRecordSet{
			Name: aws.String(fqdn),
			ResourceRecords: []*route53.ResourceRecord{{
				Value: aws.String(`"` + txtValue + `"`),
			}},
			TTL:  aws.Int64(15),
			Type: aws.String("TXT"),
		},
	}
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

func newResponder(hostedZoneId string,
	logger log.DebugLogger) (*Responder, error) {
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
	return &Responder{
		awsService:   route53.New(awsSession),
		hostedZoneId: aws.String(hostedZoneId),
		logger:       logger,
		records:      make(map[string]string),
	}, nil
}

func (r *Responder) cleanup() {
	if len(r.records) < 1 {
		return
	}
	changeBatch := make([]*route53.Change, 0, len(r.records))
	for fqdn, txtValue := range r.records {
		changeBatch = append(changeBatch, makeTXT("DELETE", fqdn, txtValue))
	}
	input := route53.ChangeResourceRecordSetsInput{
		ChangeBatch:  &route53.ChangeBatch{Changes: changeBatch},
		HostedZoneId: r.hostedZoneId,
	}
	_, err := r.awsService.ChangeResourceRecordSets(&input)
	if err != nil {
		return
	}
	r.records = make(map[string]string)
}

func (r *Responder) respond(key, value string) error {
	if r.records[key] == value {
		return nil
	}
	r.logger.Debugf(1, "publishing %s TXT=\"%s\"\n", key, value)
	input := route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{Changes: []*route53.Change{
			makeTXT("UPSERT", key, value)},
		},
		HostedZoneId: r.hostedZoneId,
	}
	output, err := r.awsService.ChangeResourceRecordSets(&input)
	if err != nil {
		return err
	}
	r.logger.Debugf(1, "waiting for change: %s to complete\n",
		*output.ChangeInfo.Id)
	err = waitForChange(r.awsService, output.ChangeInfo.Id, r.logger)
	if err != nil {
		return err
	}
	r.records[key] = value
	r.logger.Debugf(1, "change: %s completed\n", *output.ChangeInfo.Id)
	return nil
}

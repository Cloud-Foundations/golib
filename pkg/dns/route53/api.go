/*
Package route53 implements a simple DNS A record reader and writer using AWS
Route 53.
*/
package route53

import (
	"time"

	"github.com/Cloud-Foundations/golib/pkg/log"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
)

type RecordReadWriter struct {
	awsService   *route53.Route53
	hostedZoneId *string
	logger       log.DebugLogger
}

// New creates a *RecordReadWriter.
// The logger is used for logging messages.
func New(awsSession *session.Session, hostedZoneId string,
	logger log.DebugLogger) (*RecordReadWriter, error) {
	return newRecordReadWriter(awsSession, hostedZoneId, logger)
}

func (rrw *RecordReadWriter) DeleteRecords(fqdn, recType string) error {
	return rrw.deleteRecords(fqdn, recType)
}

func (rrw *RecordReadWriter) ReadRecords(fqdn, recType string) (
	[]string, time.Duration, error) {
	return rrw.readRecords(fqdn, recType)
}

func (rrw *RecordReadWriter) WriteRecords(fqdn, recType string,
	records []string, ttl time.Duration, wait bool) error {
	return rrw.writeRecords(fqdn, recType, records, ttl, wait)
}

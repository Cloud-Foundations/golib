/*
Package route53 implements a simple DNS A record reader and writer using AWS
Route 53.
*/
package route53

import (
	"net"
	"time"

	"github.com/Cloud-Foundations/golib/pkg/log"
	"github.com/aws/aws-sdk-go/service/route53"
)

type RecordReadWriter struct {
	awsService   *route53.Route53
	hostedZoneId *string
	logger       log.DebugLogger
}

// New creates a *RecordReadWriter.
// The logger is used for logging messages.
func New(hostedZoneId string,
	logger log.DebugLogger) (*RecordReadWriter, error) {
	return newRecordReadWriter(hostedZoneId, logger)
}

func (rrw *RecordReadWriter) ReadRecord(fqdn string) ([]net.IP, error) {
	return rrw.readRecord(fqdn)
}

func (rrw *RecordReadWriter) WriteRecord(fqdn string, ips []net.IP,
	ttl time.Duration) error {
	return rrw.writeRecord(fqdn, ips, ttl)
}

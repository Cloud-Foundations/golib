package caller

import (
	"context"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/Cloud-Foundations/golib/pkg/log"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
)

type Params struct {
	// Optional parameters.
	HttpClient   *http.Client
	Logger       log.DebugLogger
	urlValidator func(presignedUrl string) (*url.URL, error)
}

type cacheEntry struct {
	expires       time.Time
	normalisedArn arn.ARN
}

type Caller interface {
	GetCallerIdentity(ctx context.Context, presignedMethod string,
		presignedUrl string) (arn.ARN, error)
}

type callerT struct {
	params Params
	mutex  sync.Mutex            // Protect everything below.
	cache  map[string]cacheEntry // Key: presigned URL.
}

// Interface checks.
var _ Caller = (*callerT)(nil)

// New will create a caller for AWS STS presigned request URLs.
func New(params Params) (Caller, error) {
	return newCaller(params)
}

// GetCallerIdentity will verify if the specified URL is a valid AWS STS
// presigned URL and if so will return the corresponding caller identity.
func (c *callerT) GetCallerIdentity(ctx context.Context, presignedMethod string,
	presignedUrl string) (arn.ARN, error) {
	return c.getCallerIdentity(ctx, presignedMethod, presignedUrl)
}

package caller

import (
	"context"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Cloud-Foundations/golib/pkg/awsutil/presignauth"
	"github.com/Cloud-Foundations/golib/pkg/log/nulllogger"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
)

const (
	presignedUrlLifetime = 15 * time.Minute
)

type getCallerIdentityResult struct {
	Arn string
}

type getCallerIdentityResponse struct {
	GetCallerIdentityResult getCallerIdentityResult
}

func newCaller(params Params) (*callerT, error) {
	if params.HttpClient == nil {
		params.HttpClient = http.DefaultClient
	}
	if params.Logger == nil {
		params.Logger = nulllogger.New()
	}
	if params.urlValidator == nil {
		params.urlValidator = validateStsPresignedUrl
	}
	caller := &callerT{
		params: params,
		cache:  make(map[string]cacheEntry),
	}
	go caller.cleanupLoop()
	return caller, nil
}

func validateStsPresignedUrl(presignedUrl string) (*url.URL, error) {
	parsedPresignedUrl, err := url.Parse(presignedUrl)
	if err != nil {
		return nil, err
	}
	if parsedPresignedUrl.Scheme != "https" {
		return nil, fmt.Errorf("invalid scheme: %s", parsedPresignedUrl.Scheme)
	}
	if parsedPresignedUrl.Path != "/" {
		return nil, fmt.Errorf("invalid path: %s", parsedPresignedUrl.Path)
	}
	if !strings.HasPrefix(parsedPresignedUrl.RawQuery,
		"Action=GetCallerIdentity&") {
		return nil,
			fmt.Errorf("invalid action: %s", parsedPresignedUrl.RawQuery)
	}
	splitHost := strings.Split(parsedPresignedUrl.Host, ".")
	if len(splitHost) != 4 ||
		splitHost[0] != "sts" ||
		splitHost[2] != "amazonaws" ||
		splitHost[3] != "com" {
		return nil, fmt.Errorf("malformed presigned URL host")
	}
	return parsedPresignedUrl, nil
}

func (c *callerT) cleanupLoop() {
	for {
		time.Sleep(c.cleanupOnce())
	}
}

func (c *callerT) cleanupOnce() time.Duration {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	nextExpiration := time.Minute
	for presignedUrl, entry := range c.cache {
		if expiration := time.Until(entry.expires); expiration <= 0 {
			delete(c.cache, presignedUrl)
		} else if expiration < nextExpiration {
			nextExpiration = expiration
		}
	}
	return nextExpiration
}

func (c *callerT) getCallerIdentity(ctx context.Context, presignedMethod string,
	presignedUrl string) (arn.ARN, error) {
	if cv := c.getCallerIdentityCached(presignedUrl); cv != nil {
		return *cv, nil
	}
	if ctx == nil {
		ctx = context.TODO()
	}
	validatedUrl, err := c.params.urlValidator(presignedUrl)
	if err != nil {
		return arn.ARN{}, err
	}
	presignedUrl = validatedUrl.String()
	validateReq, err := http.NewRequest(presignedMethod, presignedUrl, nil)
	if err != nil {
		return arn.ARN{}, err
	}
	validateResp, err := c.params.HttpClient.Do(validateReq)
	if err != nil {
		return arn.ARN{}, err
	}
	defer validateResp.Body.Close()
	if validateResp.StatusCode != http.StatusOK {
		return arn.ARN{}, fmt.Errorf("verification request failed")
	}
	body, err := ioutil.ReadAll(validateResp.Body)
	if err != nil {
		return arn.ARN{}, err
	}
	var callerIdentity getCallerIdentityResponse
	if err := xml.Unmarshal(body, &callerIdentity); err != nil {
		return arn.ARN{}, err
	}
	parsedArn, err := arn.Parse(callerIdentity.GetCallerIdentityResult.Arn)
	if err != nil {
		return arn.ARN{}, err
	}
	normalisedArn, err := presignauth.NormaliseARN(parsedArn)
	if err != nil {
		return arn.ARN{}, err
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.cache[presignedUrl] = cacheEntry{
		expires:       time.Now().Add(presignedUrlLifetime),
		normalisedArn: normalisedArn,
	}
	return normalisedArn, nil
}

func (c *callerT) getCallerIdentityCached(presignedUrl string) *arn.ARN {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	entry, ok := c.cache[presignedUrl]
	if !ok {
		return nil
	}
	if time.Since(entry.expires) >= 0 {
		delete(c.cache, presignedUrl)
		return nil
	}
	return &entry.normalisedArn
}

package caller

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"testing"
)

const (
	awsTestArn                = "arn:aws:iam::accountid:role/TestMonkey"
	awsPresignedUrlBadAction  = "https://sts.a-region.amazonaws.com/?Action=BecomeRoot&Version=2011-06-15&X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=cred&X-Amz-Security-Token=token&X-Amz-SignedHeaders=host&X-Amz-Signature=sig"
	awsPresignedUrlBadDomain  = "https://sts.a-region.hackerz.com/?Action=GetCallerIdentity&Version=2011-06-15&X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=cred&X-Amz-Security-Token=token&X-Amz-SignedHeaders=host&X-Amz-Signature=sig"
	awsPresignedUrlGood       = "https://sts.a-region.amazonaws.com/?Action=GetCallerIdentity&Version=2011-06-15&X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=cred&X-Amz-Security-Token=token&X-Amz-SignedHeaders=host&X-Amz-Signature=sig"
	awsCallerIdentityResponse = `<GetCallerIdentityResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/">
  <GetCallerIdentityResult>
    <Arn>arn:aws:sts::accountid:assumed-role/TestMonkey/tester</Arn>
    <UserId>useridstuff:tester</UserId>
    <Account>accountid</Account>
  </GetCallerIdentityResult>
  <ResponseMetadata>
    <RequestId>some-uuid</RequestId>
  </ResponseMetadata>
</GetCallerIdentityResponse>
`
)

var serverCount uint

type testAwsGetCallerIdentityType struct{}

func testValidatePresignedUrl(presignedUrl string) (*url.URL, error) {
	return url.Parse(presignedUrl)
}

func (testAwsGetCallerIdentityType) ServeHTTP(w http.ResponseWriter,
	r *http.Request) {
	serverCount++
	w.Write([]byte(awsCallerIdentityResponse))
}

func TestAwsPresignedUrlValidation(t *testing.T) {
	if _, err := validateStsPresignedUrl(awsPresignedUrlBadAction); err == nil {
		t.Errorf("no error with bad action URL: %s", awsPresignedUrlBadAction)
	}
	if _, err := validateStsPresignedUrl(awsPresignedUrlBadDomain); err == nil {
		t.Errorf("no error with bad domain URL: %s", awsPresignedUrlBadDomain)
	}
	if _, err := validateStsPresignedUrl(awsPresignedUrlGood); err != nil {
		t.Error("valid URL does not validate")
	}
}

func TestAwsGetCallerIdentity(t *testing.T) {
	client, err := New(Params{urlValidator: testValidatePresignedUrl})
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "localhost:")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		err := http.Serve(listener, &testAwsGetCallerIdentityType{})
		if err != nil {
			t.Fatal(err)
		}
	}()
	testUrl := fmt.Sprintf("http://%s/", listener.Addr().String())
	callerArn, err := client.GetCallerIdentity(nil, "GET", testUrl)
	if err != nil {
		t.Fatal(err)
	}
	if callerArn.String() != awsTestArn {
		t.Errorf("expected: %s but got: %s", awsTestArn, callerArn)
	}
	// Check again to see if caching works.
	callerArn, err = client.GetCallerIdentity(nil, "GET", testUrl)
	if err != nil {
		t.Fatal(err)
	}
	if callerArn.String() != awsTestArn {
		t.Errorf("expected: %s but got: %s", awsTestArn, callerArn)
	}
	if serverCount != 1 {
		t.Errorf("serverCount expected: 1 but got: %d", serverCount)
	}
}

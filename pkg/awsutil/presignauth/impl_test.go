package presignauth

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
)

func TestBadResource(t *testing.T) {
	output, err := NormaliseARN(arn.ARN{
		Partition: "aws",
		Service:   "iam",
		AccountID: "anAccount",
		Resource:  "no-slashes",
	})
	if err == nil {
		t.Errorf("no error for bad resource, output: %v\n", output)
	}
	output, err = NormaliseARN(arn.ARN{
		Partition: "aws",
		Service:   "iam",
		AccountID: "anAccount",
		Resource:  "role/aRole",
	})
	if err == nil {
		t.Errorf("no error for bad resource, output: %v\n", output)
	}
}

func TestBadService(t *testing.T) {
	output, err := NormaliseARN(arn.ARN{
		Partition: "aws",
		Service:   "ec2",
		AccountID: "anAccount",
		Resource:  "role/SomeRole",
	})
	if err == nil {
		t.Errorf("no error for bad service, output: %v\n", output)
	}
}

func TestGoodRole(t *testing.T) {
	output, err := NormaliseARN(arn.ARN{
		Partition: "aws",
		Service:   "iam",
		AccountID: "anAccount",
		Resource:  "assumed-role/SomeRole",
	})
	if err != nil {
		t.Fatal(err)
	}
	if output.Resource != "role/SomeRole" {
		t.Errorf("unexpected role: %s", output.Resource)
	}
	output, err = NormaliseARN(arn.ARN{
		Partition: "aws",
		Service:   "sts",
		AccountID: "anAccount",
		Resource:  "assumed-role/SomeRole",
	})
	if err != nil {
		t.Fatal(err)
	}
	if output.Resource != "role/SomeRole" {
		t.Errorf("unexpected role: %s", output.Resource)
	}
}

func TestGoodRoleWithSession(t *testing.T) {
	output, err := NormaliseARN(arn.ARN{
		Partition: "aws",
		Service:   "iam",
		AccountID: "anAccount",
		Resource:  "assumed-role/SomeRole/aUser",
	})
	if err != nil {
		t.Fatal(err)
	}
	if output.Resource != "role/SomeRole" {
		t.Errorf("unexpected role: %s", output.Resource)
	}
	output, err = NormaliseARN(arn.ARN{
		Partition: "aws",
		Service:   "sts",
		AccountID: "anAccount",
		Resource:  "assumed-role/SomeRole/aUser",
	})
	if err != nil {
		t.Fatal(err)
	}
	if output.Resource != "role/SomeRole" {
		t.Errorf("unexpected role: %s", output.Resource)
	}
}

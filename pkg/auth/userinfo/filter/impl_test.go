package filter

import (
	"fmt"
	"testing"
)

type groupLister []string

var (
	inputGroups = groupLister{
		"AWS-IAM-Dev-Admin",
		"AWS-IAM-Dev-Engineer",
		"GCP-IAM-Prod-Admin",
		"GCP-IAM-Prod-Engineer",
	}
)

func (gl groupLister) GetUserGroups(username string) ([]string, error) {
	return gl, nil
}

func TestNoFilter(t *testing.T) {
	uinfo, err := NewUserGroupsFilter(inputGroups, "")
	if err != nil {
		t.Fatal(err)
	}
	groups, err := uinfo.GetUserGroups("user")
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != len(inputGroups) {
		t.Fatalf("%d output groups != %d input groups",
			len(groups), len(inputGroups))
	}
	if groups[0] != inputGroups[0] {
		t.Errorf("expected: %s, got: %s", inputGroups[0], groups[0])
	}
}

func testFilter(input, regex, expected string) error {
	uinfo, err := NewUserGroupsFilter(groupLister{input}, regex)
	if err != nil {
		return err
	}
	groups, err := uinfo.GetUserGroups("user")
	if err != nil {
		return err
	}
	if expected == "" {
		if len(groups) == 0 {
			return nil
		}
		return fmt.Errorf("expected nothing, got: %v", groups)
	}
	if len(groups) != 1 {
		return fmt.Errorf("%d output groups != 1 input group", len(groups))
	}
	if groups[0] != expected {
		return fmt.Errorf("input: %s, output: %s != expected: %s",
			input, groups[0], expected)
	}
	return nil
}

func TestFrontFilter(t *testing.T) {
	err := testFilter("AWS-IAM-Dev-Admin", "^AWS-IAM-", "Dev-Admin")
	if err != nil {
		t.Fatal(err)
	}
}

func TestMiddleFilter(t *testing.T) {
	err := testFilter("team-JUNK-name", "-JUNK", "team-name")
	if err != nil {
		t.Fatal(err)
	}
}

func TestRearFilter(t *testing.T) {
	err := testFilter("team-name-junk", "-junk$", "team-name")
	if err != nil {
		t.Fatal(err)
	}
}

func TestUnmatchedFilter(t *testing.T) {
	err := testFilter("GCP-IAM-Dev-Admin", "^AWS-IAM-", "")
	if err != nil {
		t.Fatal(err)
	}
}

package authinfo

import (
	"sync"
)

// AuthInfo contains authentication information.
type AuthInfo struct {
	AwsRole          *AwsRole
	Groups           []string
	PermittedMethods []string
	Username         string
	mutex            sync.Mutex // Protect everything below.
	groups           map[string]struct{}
	permittedMethods map[string]struct{}
}

type AwsRole struct {
	AccountId string
	ARN       string
	Name      string
}

// ListToMap is a convenience function to convert a slice of strings to a map
// of strings.
func ListToMap(list []string) map[string]struct{} {
	return listToMap(list)
}

// MapToList is a convenience function to convert a map of strings to a sorted
// slice of strings.
func MapToList(list map[string]struct{}) []string {
	return mapToList(list)
}

// CheckGroup will return true if the specified group is present in the list.
// It uses an O(1) lookup.
func (ai *AuthInfo) CheckGroup(group string) bool {
	return ai.checkGroup(group)
}

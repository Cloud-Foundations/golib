package authinfo

import (
	"context"
	"io"
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

// ContextWithAuthInfo returns a copy of a context with authentication
// information added.
func ContextWithAuthInfo(ctx context.Context,
	authInfo AuthInfo) context.Context {
	return contextWithAuthInfo(ctx, authInfo)
}

// GetAuthInfoFromContext will return authentication information from a
// context, if available.
func GetAuthInfoFromContext(ctx context.Context) *AuthInfo {
	return getAuthInfoFromContext(ctx)
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

// Write will write the authentication information in a human-readable
// format. Each line will be preceeded by a prefix and succeeded with a postfix.
// The specified space string will be used for further indentation.
func (ai *AuthInfo) Write(writer io.Writer,
	prefix, space, postfix string) error {
	return ai.write(writer, prefix, space, postfix)
}

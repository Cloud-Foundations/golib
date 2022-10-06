package authinfo

import (
	"context"
	"fmt"
	"io"
	"sort"
)

type contextKey int

var authContextKey = new(contextKey)

func contextWithAuthInfo(ctx context.Context,
	authInfo AuthInfo) context.Context {
	return context.WithValue(ctx, authContextKey, authInfo)
}

func formatList(list []string, prefix, space, postfix string) []string {
	var formattedList []string
	for _, entry := range list {
		formattedList = append(formattedList,
			fmt.Sprintf("%s%s%s%s%s\n",
				prefix, space, space, entry, postfix))
	}
	return formattedList
}

func getAuthInfoFromContext(ctx context.Context) *AuthInfo {
	if val := ctx.Value(authContextKey); val != nil {
		if authInfo, ok := val.(AuthInfo); ok {
			return &authInfo
		}
	}
	return nil
}

func listToMap(list []string) map[string]struct{} {
	mapList := make(map[string]struct{}, len(list))
	for _, entry := range list {
		mapList[entry] = struct{}{}
	}
	return mapList
}

func mapToList(list map[string]struct{}) []string {
	var sortedList []string
	for entry := range list {
		sortedList = append(sortedList, entry)
	}
	sort.Strings(sortedList)
	return sortedList
}

func writeList(writer io.Writer, list []string, prefix, space, postfix string) {
	for _, line := range formatList(list, prefix, space, postfix) {
		fmt.Fprint(writer, line)
	}
}

func (ai *AuthInfo) checkGroup(group string) bool {
	ai.mutex.Lock()
	defer ai.mutex.Unlock()
	if len(ai.Groups) != len(ai.groups) {
		ai.groups = listToMap(ai.Groups)
	}
	_, ok := ai.groups[group]
	return ok
}

func (ai *AuthInfo) write(writer io.Writer,
	prefix, space, postfix string) error {
	if ai == nil {
		fmt.Fprintf(writer, "%sNo authentication information%s\n",
			prefix, postfix)
		return nil
	}
	if ai.Username != "" {
		fmt.Fprintf(writer, "%sUsername: %s%s\n",
			prefix, ai.Username, postfix)
	} else if ai.AwsRole != nil {
		fmt.Fprintf(writer, "%sAWS role: %s in account: %s (ARN=%s)%s\n",
			prefix, ai.AwsRole.Name, ai.AwsRole.AccountId,
			ai.AwsRole.ARN, postfix)
	} else {
		fmt.Fprintf(writer, "%sUnknown principal%s\n", prefix, postfix)
	}
	if len(ai.Groups) > 0 {
		fmt.Fprintf(writer, "%sGroup list:%s\n", prefix, postfix)
		writeList(writer, ai.Groups, prefix, space, postfix)
	} else {
		fmt.Fprintf(writer, "%sNo group memberships%s\n", prefix, postfix)
	}
	if len(ai.PermittedMethods) > 0 {
		fmt.Fprintf(writer, "%sPermitted methods:%s\n", prefix, postfix)
		writeList(writer, ai.PermittedMethods, prefix, space, postfix)
	} else {
		fmt.Fprintf(writer, "%sNo methods are permitted%s\n",
			prefix, postfix)
	}
	if !ai.Expires.IsZero() {
		fmt.Fprintf(writer, "%sAuthentication information expires: %s%s\n",
			prefix, ai.Expires.Local(), postfix)
	}
	return nil
}

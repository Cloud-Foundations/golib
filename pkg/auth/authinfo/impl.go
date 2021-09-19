package authinfo

import (
	"context"
	"sort"
)

type contextKey int

var authContextKey = new(contextKey)

func contextWithAuthInfo(ctx context.Context,
	authInfo AuthInfo) context.Context {
	return context.WithValue(ctx, authContextKey, authInfo)
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

func (ai *AuthInfo) checkGroup(group string) bool {
	ai.mutex.Lock()
	defer ai.mutex.Unlock()
	if len(ai.Groups) != len(ai.groups) {
		ai.groups = listToMap(ai.Groups)
	}
	_, ok := ai.groups[group]
	return ok
}

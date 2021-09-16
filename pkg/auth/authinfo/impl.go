package authinfo

import "sort"

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

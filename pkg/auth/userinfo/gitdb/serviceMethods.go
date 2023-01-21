package gitdb

import (
	"sort"
)

func (uinfo *UserInfo) getGroupServiceMethods(groupname string) (
	[]string, error) {
	uinfo.rwMutex.RLock()
	defer uinfo.rwMutex.RUnlock()
	serviceMethods := uinfo.groupMethods[groupname]
	if len(serviceMethods) < 1 {
		return nil, nil
	}
	smStrings := make([]string, 0, len(serviceMethods))
	for _, serviceMethod := range serviceMethods {
		smStrings = append(smStrings,
			serviceMethod.service+"."+serviceMethod.method)
	}
	return smStrings, nil
}

func (uinfo *UserInfo) getUserServiceMethods(username string) (
	[]string, error) {
	serviceMethods := uinfo.getUserServiceMethodsMap(username)
	smStrings := make([]string, 0, len(serviceMethods))
	for serviceMethod := range serviceMethods {
		smStrings = append(smStrings,
			serviceMethod.service+"."+serviceMethod.method)
	}
	sort.Strings(smStrings)
	return smStrings, nil
}

func (uinfo *UserInfo) getUserServiceMethodsMap(
	username string) map[serviceMethod]struct{} {
	serviceMethods := make(map[serviceMethod]struct{})
	uinfo.rwMutex.RLock()
	defer uinfo.rwMutex.RUnlock()
	for groupname := range uinfo.groupsPerUser[username] {
		for _, serviceMethod := range uinfo.groupMethods[groupname] {
			serviceMethods[serviceMethod] = struct{}{}
		}
	}
	return serviceMethods
}

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
	if methods, ok := serviceMethods["*"]; ok {
		if _, ok := methods["*"]; ok {
			return []string{"*.*"}, nil
		}
		for method := range methods {
			smStrings = append(smStrings, "*."+method)
		}
	}
	for service, methods := range serviceMethods {
		if _, ok := methods["*"]; ok {
			smStrings = append(smStrings, service+".*")
		} else {
			for method := range methods {
				if _, ok := serviceMethods["*"][method]; !ok {
					smStrings = append(smStrings, service+"."+method)
				}
			}
		}
	}
	sort.Strings(smStrings)
	return smStrings, nil
}

// getUserServiceMethodsMap returns map[service-name]map[method-name]struct{}
func (uinfo *UserInfo) getUserServiceMethodsMap(
	username string) map[string]map[string]struct{} {
	serviceMethods := make(map[string]map[string]struct{})
	uinfo.rwMutex.RLock()
	defer uinfo.rwMutex.RUnlock()
	for groupname := range uinfo.groupsPerUser[username] {
		for _, serviceMethod := range uinfo.groupMethods[groupname] {
			methodsMap := serviceMethods[serviceMethod.service]
			if methodsMap == nil {
				methodsMap = make(map[string]struct{})
				serviceMethods[serviceMethod.service] = methodsMap
			}
			serviceMethods[serviceMethod.service][serviceMethod.method] =
				struct{}{}
		}
	}
	return serviceMethods
}

package filter

import (
	"regexp"

	"github.com/Cloud-Foundations/golib/pkg/auth/userinfo"
)

func newUserGroupsFilter(userGroups userinfo.UserGroupsGetter,
	regex string) (userinfo.UserGroupsGetter, error) {
	if regex == "" {
		return userGroups, nil
	}
	if re, err := regexp.Compile(regex); err != nil {
		return nil, err
	} else {
		return &UserGroupsInfo{
			re:               re,
			userGroupsGetter: userGroups,
		}, nil
	}
}

func newUserInfoFilter(userInfo userinfo.UserInfo,
	groupFilter string) (userinfo.UserInfo, error) {
	if groupFilter == "" {
		return userInfo, nil
	}
	if re, err := regexp.Compile(groupFilter); err != nil {
		return nil, err
	} else {
		return &UserInfo{
			UserGroupsInfo: UserGroupsInfo{
				re:               re,
				userGroupsGetter: userInfo,
			},
			UserInfo: userInfo,
		}, nil
	}
}

func (uinfo *UserGroupsInfo) getUserGroups(username string) ([]string, error) {
	groups, err := uinfo.userGroupsGetter.GetUserGroups(username)
	if err != nil {
		return nil, err
	}
	outputGroups := make([]string, 0, len(groups))
	for _, group := range groups {
		if uinfo.re.MatchString(group) {
			output := uinfo.re.ReplaceAllString(group, "")
			if output != "" {
				outputGroups = append(outputGroups, output)
			}
		}
	}
	return outputGroups, nil
}

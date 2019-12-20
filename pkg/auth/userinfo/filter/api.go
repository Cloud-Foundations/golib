package filter

import (
	"regexp"

	"github.com/Cloud-Foundations/golib/pkg/auth/userinfo"
)

type UserGroupsInfo struct {
	re               *regexp.Regexp
	userGroupsGetter userinfo.UserGroupsGetter
}

type UserInfo struct {
	UserGroupsInfo
	userinfo.UserInfo
}

func NewUserGroupsFilter(userGroups userinfo.UserGroupsGetter,
	regex string) (userinfo.UserGroupsGetter, error) {
	return newUserGroupsFilter(userGroups, regex)
}

func (uinfo *UserGroupsInfo) GetUserGroups(username string) ([]string, error) {
	return uinfo.getUserGroups(username)
}

func NewUserInfoFilter(userInfo userinfo.UserInfo,
	groupFilter string) (userinfo.UserInfo, error) {
	return newUserInfoFilter(userInfo, groupFilter)
}

func (uinfo *UserInfo) GetUserGroups(username string) ([]string, error) {
	return uinfo.UserGroupsInfo.GetUserGroups(username)
}

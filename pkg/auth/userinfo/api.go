package userinfo

// UserGroupsGetter is the interface that wraps the GetUserGroups method.
//
// GetUserGroups gets the groups that the user specified by username is a member
// of.
type UserGroupsGetter interface {
	GetUserGroups(username string) ([]string, error)
}

// UserInfo is the interface that wraps multiple methods which yield information
// about a user.
type UserInfo interface {
	UserGroupsGetter
}

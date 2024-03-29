package userinfo

// GroupServiceMethodsGetter is the interface that wraps the
// GetGroupServiceMethods method.
type GroupServiceMethodsGetter interface {
	GetGroupServiceMethods(groupname string) ([]string, error)
}

// UserGroupsGetter is the interface that wraps the GetUserGroups method.
//
// GetUserGroups gets the groups that the user specified by username is a member
// of.
type UserGroupsGetter interface {
	GetUserGroups(username string) ([]string, error)
}

// UsersInGroupGetter is the interface that wraps the GetUsersInGroup method.
//
// GetUsersInGroup gets the list of users which are members of the group.
type UsersInGroupGetter interface {
	GetUsersInGroup(groupname string) ([]string, error)
}

// UserInfo is the interface that wraps multiple methods which yield information
// about a user.
type UserInfo interface {
	UserGroupsGetter
}

// UserServiceMethodsGetter is the interface that wraps the
// GetUserServiceMethods method.
type UserServiceMethodsGetter interface {
	GetUserServiceMethods(username string) ([]string, error)
}

package gitdb

import (
	"sync"
	"time"

	"github.com/Cloud-Foundations/golib/pkg/auth/userinfo"
	"github.com/Cloud-Foundations/golib/pkg/git/repowatch"
	"github.com/Cloud-Foundations/golib/pkg/log"
)

type Config struct {
	repowatch.Config `yaml:",inline"`
}

type Params struct {
	repowatch.Params
}

type serviceMethod struct {
	method  string
	service string
}

type UserInfo struct {
	logger        log.DebugLogger
	rwMutex       sync.RWMutex                   // Protect everything below.
	groupsPerUser map[string]map[string]struct{} // K: username, V: groups.
	groupMethods  map[string][]serviceMethod     // K: groupname.
	usersPerGroup map[string]map[string]struct{} // K: groupname, V: usernames.
}

// Interface checks to ensure we don't regress.
var (
	_ userinfo.GroupServiceMethodsGetter = (*UserInfo)(nil)
	_ userinfo.UserInfo                  = (*UserInfo)(nil)
	_ userinfo.UserServiceMethodsGetter  = (*UserInfo)(nil)
)

// New is a deprecated interface. Use New2 instead.
func New(repositoryURL, branch, localRepositoryDir string,
	checkInterval time.Duration, logger log.DebugLogger) (
	*UserInfo, error) {
	return newDB(Config{Config: repowatch.Config{
		Branch:                   branch,
		CheckInterval:            checkInterval,
		LocalRepositoryDirectory: localRepositoryDir,
		RepositoryURL:            repositoryURL,
	}},
		Params{Params: repowatch.Params{
			Logger: logger,
		}},
	)
}

// NewWithConfig is a deprecated interface. Use New2 instead.
func NewWithConfig(config Config, logger log.DebugLogger) (*UserInfo, error) {
	return newDB(config, Params{Params: repowatch.Params{Logger: logger}})
}

// New opens a *UserInfo database using Git as the backing store. It will
// periodically pull from the remote repository specified by
// config.RepositoryURL and cache a local copy in the
// config.LocalRepositoryDirectory. If config.RepositoryURL is empty then
// only the local repository is used.
// The specified config.Branch is read to extract the database.
// The databse is checked every config.CheckInterval for updates.
// Any problems with fetching or updating the database are sent to the logger.
func New2(config Config, params Params) (*UserInfo, error) {
	return newDB(config, params)
}

func (uinfo *UserInfo) GetGroups() ([]string, error) {
	return uinfo.getGroups()
}

func (uinfo *UserInfo) GetGroupServiceMethods(groupname string) (
	[]string, error) {
	return uinfo.getGroupServiceMethods(groupname)
}

func (uinfo *UserInfo) GetUserGroups(username string) ([]string, error) {
	return uinfo.getUserGroups(username)
}

func (uinfo *UserInfo) GetUsersInGroup(groupname string) ([]string, error) {
	return uinfo.getUsersInGroup(groupname)
}

func (uinfo *UserInfo) GetUsersInGroups() ([]string, error) {
	return uinfo.getUsersInGroups()
}

func (uinfo *UserInfo) GetUserServiceMethods(username string) (
	[]string, error) {
	return uinfo.getUserServiceMethods(username)
}

func (uinfo *UserInfo) TestUserInGroup(username, groupname string) bool {
	return uinfo.testUserInGroup(username, groupname)
}

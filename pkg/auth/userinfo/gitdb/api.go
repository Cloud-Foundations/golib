package gitdb

import (
	"sync"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/repowatch"
	"github.com/Cloud-Foundations/golib/pkg/log"
)

type Config struct {
	repowatch.Config `yaml:",inline"`
}

type UserInfo struct {
	logger        log.DebugLogger
	rwMutex       sync.RWMutex                   // Protect everything below.
	groupsPerUser map[string]map[string]struct{} // K: username, V: groups.
}

func New(repositoryURL, branch, localRepositoryDir string,
	checkInterval time.Duration, logger log.DebugLogger) (
	*UserInfo, error) {
	return newDB(Config{Config: repowatch.Config{
		Branch:                   branch,
		CheckInterval:            checkInterval,
		LocalRepositoryDirectory: localRepositoryDir,
		RepositoryURL:            repositoryURL,
	}}, logger)
}

func NewWithConfig(config Config, logger log.DebugLogger) (*UserInfo, error) {
	return newDB(config, logger)
}

func (uinfo *UserInfo) GetUserGroups(username string) ([]string, error) {
	return uinfo.getUserGroups(username)
}

func (uinfo *UserInfo) GetUsersInGroups() ([]string, error) {
	return uinfo.getUsersInGroups()
}

func (uinfo *UserInfo) TestUserInGroup(username, groupname string) bool {
	return uinfo.testUserInGroup(username, groupname)
}

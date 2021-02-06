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
	usersPerGroup map[string]map[string]struct{} // K: groupname, V: usernames.
}

// New opens a *UserInfo database using Git as the backing store. It will
// periodically pull from the remote repository specified by repositoryURL and
// cache a local copy in the localRepositoryDir. If repositoryURL is empty then
// only the local repository is used.
// The specified branch is read to extract the database.
// The databse is checked every checkInterval for updates.
// Any problems with fetching or updating the database are sent to the logger.
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

func (uinfo *UserInfo) GetGroups() ([]string, error) {
	return uinfo.getGroups()
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

func (uinfo *UserInfo) TestUserInGroup(username, groupname string) bool {
	return uinfo.testUserInGroup(username, groupname)
}

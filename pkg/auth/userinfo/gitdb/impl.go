package gitdb

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/Cloud-Foundations/Dominator/lib/decoders"
	"github.com/Cloud-Foundations/Dominator/lib/repowatch"
	"github.com/Cloud-Foundations/golib/pkg/log"
)

var (
	repoRE = regexp.MustCompile(".*@(.*)[.]git$")
)

type groupType struct {
	Email        []string `json:",omitempty"`
	GroupMembers []string `json:",omitempty"`
	Name         string   `json:",omitempty"`
	UserMembers  []string `json:",omitempty"`
	processing   bool
	users        map[string]struct{} // Includes sub-groups.
}

type loadStateType struct {
	groupsPerUser map[string]map[string]struct{}
	groupsMap     map[string]*groupType
	logger        log.DebugLogger
}

func addUserList(addTo, addFrom map[string]struct{}) {
	for user := range addFrom {
		addTo[user] = struct{}{}
	}
}

func newDB(config Config, logger log.DebugLogger) (*UserInfo, error) {
	if config.Branch != "" && config.Branch != "master" {
		return nil, errors.New("non-master branch not supported")
	}
	metricsSubdir := config.LocalRepositoryDirectory
	if config.RepositoryURL != "" {
		metricsSubdir = repoRE.ReplaceAllString(config.RepositoryURL, "$1")
	}
	directoryChannel, err := repowatch.WatchWithConfig(config.Config,
		filepath.Join("userinfo/gitdb", metricsSubdir), logger)
	if err != nil {
		return nil, err
	}
	userInfo := &UserInfo{logger: logger}
	// Consume initial notification to ensure DB is populated before returning.
	if err := userInfo.loadDatabase(<-directoryChannel); err != nil {
		userInfo.logger.Println(err)
	}
	go userInfo.handleNotifications(directoryChannel)
	return userInfo, nil
}

func (ls *loadStateType) loadDirectory(dirname string) error {
	var permittedGroupsExpressions []string
	err := decoders.FindAndDecodeFile(
		filepath.Join(dirname, "permitted-groups"),
		&permittedGroupsExpressions)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	permittedGroupsREs := make([]*regexp.Regexp, 0,
		len(permittedGroupsExpressions))
	for _, regex := range permittedGroupsExpressions {
		if re, err := regexp.Compile("^" + regex + "$"); err != nil {
			return fmt.Errorf("error RE compiling: \"%s\": %s", regex, err)
		} else {
			permittedGroupsREs = append(permittedGroupsREs, re)
		}
	}
	var groups []*groupType
	err = decoders.FindAndDecodeFile(filepath.Join(dirname, "groups"), &groups)
	if err != nil {
		if !os.IsNotExist(err) {
			ls.logger.Printf("%s: ignoring", err)
		}
		return nil
	}
	for _, group := range groups {
		permitted := false
		for _, re := range permittedGroupsREs {
			if re.MatchString(group.Name) {
				permitted = true
				break
			}
		}
		if permitted {
			if _, ok := ls.groupsMap[group.Name]; !ok {
				ls.groupsMap[group.Name] = group
				// Process direct memberships now.
				for _, user := range group.UserMembers {
					if gtable, ok := ls.groupsPerUser[user]; !ok {
						ls.groupsPerUser[user] = map[string]struct{}{
							group.Name: {},
						}
					} else {
						gtable[group.Name] = struct{}{}
					}
				}
			} else {
				ls.logger.Printf("%s: %s group: \"%s\" already defined",
					dirname, group.Name)
			}
		} else {
			ls.logger.Printf("group: \"%s\" not permitted in: %s\n",
				group.Name, dirname)
		}
	}
	return nil
}

func (ls *loadStateType) processGroup(group *groupType) {
	if group.users != nil {
		return
	}
	if group.processing {
		ls.logger.Printf("group: \"%s\" is part of a loop, skipping\n",
			group.Name)
		return
	}
	group.processing = true
	defer func() { group.processing = false }()
	userList := make(map[string]struct{})
	for _, memberGroupName := range group.GroupMembers {
		if memberGroup, ok := ls.groupsMap[memberGroupName]; !ok {
			ls.logger.Printf("%s references group that does not exist: %s\n",
				group.Name, memberGroupName)
		} else {
			ls.processGroup(memberGroup)
			addUserList(userList, memberGroup.users)
		}
	}
	for _, user := range group.UserMembers {
		userList[user] = struct{}{}
	}
	for user := range userList {
		ls.groupsPerUser[user][group.Name] = struct{}{}
	}
	group.users = userList
}

func (ls *loadStateType) walkDirectory(dirname string) error {
	if err := ls.loadDirectory(dirname); err != nil {
		return err
	}
	directory, err := os.Open(dirname)
	if err != nil {
		return err
	}
	filenames, err := directory.Readdirnames(-1)
	directory.Close()
	if err != nil {
		return err
	}
	for _, filename := range filenames {
		if filename == ".git" {
			continue
		}
		pathname := filepath.Join(dirname, filename)
		if fi, err := os.Stat(pathname); err != nil {
			return err
		} else if fi.IsDir() {
			if err := ls.walkDirectory(pathname); err != nil {
				return err
			}
		}
	}
	return nil
}

func (uinfo *UserInfo) getGroups() ([]string, error) {
	uinfo.rwMutex.RLock()
	defer uinfo.rwMutex.RUnlock()
	groups := make([]string, 0, len(uinfo.usersPerGroup))
	for group := range uinfo.usersPerGroup {
		groups = append(groups, group)
	}
	return groups, nil
}

func (uinfo *UserInfo) getUserGroups(username string) ([]string, error) {
	uinfo.rwMutex.RLock()
	groupsMap := uinfo.groupsPerUser[username]
	uinfo.rwMutex.RUnlock()
	groups := make([]string, 0, len(groupsMap))
	for group := range groupsMap {
		groups = append(groups, group)
	}
	return groups, nil
}

func (uinfo *UserInfo) getUsersInGroup(groupname string) ([]string, error) {
	uinfo.rwMutex.RLock()
	group, ok := uinfo.usersPerGroup[groupname]
	uinfo.rwMutex.RUnlock()
	if !ok {
		return nil, fmt.Errorf("group: %s not found", groupname)
	}
	usernames := make([]string, 0, len(group))
	for username := range group {
		usernames = append(usernames, username)
	}
	return usernames, nil
}

func (uinfo *UserInfo) getUsersInGroups() ([]string, error) {
	uinfo.rwMutex.RLock()
	defer uinfo.rwMutex.RUnlock()
	usernames := make([]string, 0, len(uinfo.groupsPerUser))
	for username := range uinfo.groupsPerUser {
		usernames = append(usernames, username)
	}
	return usernames, nil
}

func (uinfo *UserInfo) handleNotifications(directoryChannel <-chan string) {
	for dirname := range directoryChannel {
		if err := uinfo.loadDatabase(dirname); err != nil {
			uinfo.logger.Println(err)
		}
	}
}

func (uinfo *UserInfo) loadDatabase(dirname string) error {
	loadState := &loadStateType{
		groupsPerUser: make(map[string]map[string]struct{}),
		groupsMap:     make(map[string]*groupType),
		logger:        uinfo.logger,
	}
	if err := loadState.walkDirectory(dirname); err != nil {
		return err
	}
	usersPerGroup := make(map[string]map[string]struct{})
	for _, group := range loadState.groupsMap {
		loadState.processGroup(group)
		usersPerGroup[group.Name] = group.users
	}
	uinfo.rwMutex.Lock()
	defer uinfo.rwMutex.Unlock()
	uinfo.groupsPerUser = loadState.groupsPerUser
	uinfo.usersPerGroup = usersPerGroup
	return nil
}

func (uinfo *UserInfo) testUserInGroup(username, groupname string) bool {
	uinfo.rwMutex.RLock()
	defer uinfo.rwMutex.RUnlock()
	if groups, ok := uinfo.groupsPerUser[username]; !ok {
		return false
	} else {
		_, inGroup := groups[groupname]
		return inGroup
	}
}

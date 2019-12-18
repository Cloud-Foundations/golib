package gitdb

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/decoders"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/repowatch"
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
}

func addUserList(addTo, addFrom map[string]struct{}) {
	for user := range addFrom {
		addTo[user] = struct{}{}
	}
}

func loadDirectory(dirname string, loadState *loadStateType,
	logger log.DebugLogger) error {
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
			logger.Printf("%s: ignoring", err)
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
			if _, ok := loadState.groupsMap[group.Name]; !ok {
				loadState.groupsMap[group.Name] = group
				// Process direct memberships now.
				for _, user := range group.UserMembers {
					if gtable, ok := loadState.groupsPerUser[user]; !ok {
						loadState.groupsPerUser[user] = map[string]struct{}{
							group.Name: {},
						}
					} else {
						gtable[group.Name] = struct{}{}
					}
				}
			} else {
				logger.Printf("%s: %s group: \"%s\" already defined",
					dirname, group.Name)
			}
		} else {
			logger.Printf("group: \"%s\" not permitted in: %s\n",
				group.Name, dirname)
		}
	}
	return nil
}

func newDB(repositoryURL, branch, localRepositoryDir string,
	checkInterval time.Duration, logger log.DebugLogger) (*UserInfo, error) {
	if branch != "" && branch != "master" {
		return nil, errors.New("non-master branch not supported")
	}
	metricsSubdir := localRepositoryDir
	if repositoryURL != "" {
		metricsSubdir = repoRE.ReplaceAllString(repositoryURL, "$1")
	}
	directoryChannel, err := repowatch.Watch(repositoryURL,
		localRepositoryDir, checkInterval,
		filepath.Join("userinfo/gitdb", metricsSubdir),
		logger)
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

func walkDirectory(dirname string, loadState *loadStateType,
	logger log.DebugLogger) error {
	if err := loadDirectory(dirname, loadState, logger); err != nil {
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
			if err := walkDirectory(pathname, loadState, logger); err != nil {
				return err
			}
		}
	}
	return nil
}

func (loadState *loadStateType) processGroup(group *groupType,
	logger log.DebugLogger) {
	if group.users != nil {
		return
	}
	if group.processing {
		logger.Printf("group: \"%s\" is part of a loop, skipping\n",
			group.Name)
		return
	}
	group.processing = true
	defer func() { group.processing = false }()
	userList := make(map[string]struct{})
	for _, memberGroupName := range group.GroupMembers {
		if memberGroup, ok := loadState.groupsMap[memberGroupName]; !ok {
			logger.Printf("%s references group that does not exist: %s\n",
				group.Name, memberGroupName)
		} else {
			loadState.processGroup(memberGroup, logger)
			addUserList(userList, memberGroup.users)
		}
	}
	for _, user := range group.UserMembers {
		userList[user] = struct{}{}
	}
	for user := range userList {
		loadState.groupsPerUser[user][group.Name] = struct{}{}
	}
	group.users = userList
}

func (uinfo *UserInfo) getUserGroups(username string) ([]string, error) {
	uinfo.rwMutex.RLock()
	groupsMap := uinfo.groupsPerUser[username]
	groups := make([]string, 0, len(groupsMap))
	for group := range groupsMap {
		groups = append(groups, group)
	}
	uinfo.rwMutex.RUnlock()
	return groups, nil
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
	}
	if err := walkDirectory(dirname, loadState, uinfo.logger); err != nil {
		return err
	}
	for _, group := range loadState.groupsMap {
		loadState.processGroup(group, uinfo.logger)
	}
	uinfo.rwMutex.Lock()
	defer uinfo.rwMutex.Unlock()
	uinfo.groupsPerUser = loadState.groupsPerUser
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

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/golib/pkg/auth/userinfo/gitdb"
)

var divider = []byte("======================================================\n")

func showUserGroupsSubcommand(args []string, logger log.DebugLogger) error {
	if err := showUserGroups(os.Stdout, args[0], args[1], logger); err != nil {
		return fmt.Errorf("Error showing groups for user: %s: %s", args[1], err)
	}
	return nil
}

func showUserGroups(writer io.Writer, source, username string,
	logger log.DebugLogger) error {
	var tmpdir string
	fi, err := os.Stat(source)
	if err == nil && fi.IsDir() {
		tmpdir = source
		source = ""
	} else {
		tmpdir, err = ioutil.TempDir("", "userinfo")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmpdir)
	}
	db, err := gitdb.New(source, "", tmpdir, time.Hour, logger)
	if err != nil {
		return err
	}
	if username != "" {
		if groups, err := db.GetUserGroups(username); err != nil {
			return err
		} else {
			showLine(writer, username, groups)
			return nil
		}
	}
	usernames, err := db.GetUsersInGroups()
	if err != nil {
		return err
	}
	sort.Strings(usernames)
	usersPerGroup := make(map[string]map[string]struct{})
	for _, username := range usernames {
		if groups, err := db.GetUserGroups(username); err != nil {
			return err
		} else {
			showLine(writer, username, groups)
			for _, group := range groups {
				if usersInGroup, ok := usersPerGroup[group]; !ok {
					usersPerGroup[group] = map[string]struct{}{username: {}}
				} else {
					usersInGroup[username] = struct{}{}
				}
			}
		}
	}
	writer.Write(divider)
	groups := make([]string, 0, len(usersPerGroup))
	for group := range usersPerGroup {
		groups = append(groups, group)
	}
	sort.Strings(groups)
	for _, group := range groups {
		showLine(writer, group, mapToSlice(usersPerGroup[group]))
	}
	return nil
}

func mapToSlice(list map[string]struct{}) []string {
	retval := make([]string, 0, len(list))
	for entry := range list {
		retval = append(retval, entry)
	}
	return retval
}

func showLine(writer io.Writer, key string, values []string) {
	sort.Strings(values)
	fmt.Fprint(writer, key+":")
	for _, value := range values {
		fmt.Fprint(writer, " ", value)
	}
	fmt.Fprintln(writer)
}

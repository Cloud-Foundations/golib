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

func listUsersSubcommand(args []string, logger log.DebugLogger) error {
	if err := listUsers(os.Stdout, args[0], logger); err != nil {
		return fmt.Errorf("Error listing users from: %s: %s", args[0], err)
	}
	return nil
}

func listUsers(writer io.Writer, source string, logger log.DebugLogger) error {
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
	usernames, err := db.GetUsersInGroups()
	if err != nil {
		return err
	}
	sort.Strings(usernames)
	for _, username := range usernames {
		fmt.Fprintln(writer, username)
	}
	return nil
}

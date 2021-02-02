package main

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func listUsersSubcommand(args []string, logger log.DebugLogger) error {
	if err := listUsers(os.Stdout, args[0], logger); err != nil {
		return fmt.Errorf("Error listing users from: %s: %s", args[0], err)
	}
	return nil
}

func listUsers(writer io.Writer, source string, logger log.DebugLogger) error {
	db, err := getDB(source, logger)
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

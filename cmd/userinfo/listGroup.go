package main

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func listGroupSubcommand(args []string, logger log.DebugLogger) error {
	if err := listGroup(os.Stdout, args[0], args[1], logger); err != nil {
		return fmt.Errorf("Error listing group: %s", err)
	}
	return nil
}

func listGroup(writer io.Writer, source, groupname string,
	logger log.DebugLogger) error {
	db, err := getDB(source, logger)
	if err != nil {
		return err
	}
	usernames, err := db.GetUsersInGroup(groupname)
	if err != nil {
		return err
	}
	sort.Strings(usernames)
	for _, username := range usernames {
		fmt.Fprintln(writer, username)
	}
	return nil
}

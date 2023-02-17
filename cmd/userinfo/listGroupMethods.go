package main

import (
	"fmt"
	"io"
	"os"

	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func listGroupMethodsSubcommand(args []string, logger log.DebugLogger) error {
	if err := listGroupSMs(os.Stdout, args[0], args[1], logger); err != nil {
		return fmt.Errorf("Error listing group methods: %s", err)
	}
	return nil
}

func listGroupSMs(writer io.Writer, source, groupname string,
	logger log.DebugLogger) error {
	db, err := getDB(source, logger)
	if err != nil {
		return err
	}
	serviceMethods, err := db.GetGroupServiceMethods(groupname)
	if err != nil {
		return err
	}
	for _, serviceMethod := range serviceMethods {
		fmt.Fprintln(writer, serviceMethod)
	}
	return nil
}

package main

import (
	"fmt"
	"io"
	"os"

	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func listUserMethodsSubcommand(args []string, logger log.DebugLogger) error {
	if err := listUserSMs(os.Stdout, args[0], args[1], logger); err != nil {
		return fmt.Errorf("Error listing user methods: %s", err)
	}
	return nil
}

func listUserSMs(writer io.Writer, source, username string,
	logger log.DebugLogger) error {
	db, err := getDB(source, logger)
	if err != nil {
		return err
	}
	serviceMethods, err := db.GetUserServiceMethods(username)
	if err != nil {
		return err
	}
	for _, serviceMethod := range serviceMethods {
		fmt.Fprintln(writer, serviceMethod)
	}
	return nil
}

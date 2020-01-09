package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func diffUsersGroupsSubcommand(args []string, logger log.DebugLogger) error {
	if err := diffUsersGroups(args[0], args[1], args[2], logger); err != nil {
		return fmt.Errorf("Error diffing groups: %s", err)
	}
	return nil
}

func diffUsersGroups(difftool, source1, source2 string,
	logger log.DebugLogger) error {
	file1, err := ioutil.TempFile("", "userinfo.source1.")
	if err != nil {
		return err
	}
	defer file1.Close()
	defer os.Remove(file1.Name())
	file2, err := ioutil.TempFile("", "userinfo.source2.")
	if err != nil {
		return err
	}
	defer file2.Close()
	defer os.Remove(file2.Name())
	if err := showUserGroups(file1, source1, "", logger); err != nil {
		return err
	}
	if err := showUserGroups(file2, source2, "", logger); err != nil {
		return err
	}
	cmd := exec.Command(difftool, file1.Name(), file2.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

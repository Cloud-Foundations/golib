package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/lib/flags/commands"
	"github.com/Cloud-Foundations/Dominator/lib/flags/loadflags"
	"github.com/Cloud-Foundations/Dominator/lib/log/cmdlogger"
)

var (
	awsSecretId = flag.String("awsSecretId", "",
		"If specified, fetch the SSH key from the AWS secret object")
	ignoreErrors = flag.Bool("ignoreErrors", false,
		"If true, ignore errors in the DB")
)

func printUsage() {
	w := flag.CommandLine.Output()
	fmt.Fprintln(w, "Usage: userinfo [flags...] command [args...]")
	fmt.Fprintln(w, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(w, "Commands:")
	commands.PrintCommands(w, subcommands)
}

var subcommands = []commands.Command{
	{"diff-users-groups", "difftool source1 source2", 3, 3,
		diffUsersGroupsSubcommand},
	{"list-group", "source group", 2, 2, listGroupSubcommand},
	{"list-users", "source", 1, 1, listUsersSubcommand},
	{"show-user-groups", "source username", 2, 2, showUserGroupsSubcommand},
}

func doMain() int {
	if err := loadflags.LoadForCli("userinfo"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	flag.Usage = printUsage
	flag.Parse()
	if flag.NArg() < 1 {
		printUsage()
		return 3
	}
	logger := cmdlogger.New()
	return commands.RunCommands(subcommands, printUsage, logger)
}

func main() {
	os.Exit(doMain())
}

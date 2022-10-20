package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/decoders"
	"github.com/Cloud-Foundations/Dominator/lib/flags/commands"
	"github.com/Cloud-Foundations/Dominator/lib/flags/loadflags"
	"github.com/Cloud-Foundations/golib/pkg/loadbalancing/dnslb/config"
	"github.com/Cloud-Foundations/golib/pkg/log/cmdlogger"

	"gopkg.in/yaml.v2"
)

var (
	blockDuration = flag.Duration("blockDuration", time.Minute*15,
		"Duration to block")
	configFile = flag.String("configFile", "",
		"Name of file containing configuration")

	cfgData config.Config
)

func init() {
	decoders.RegisterDecoder(".yml", yamlDecoderGenerator)
}

func printUsage() {
	w := flag.CommandLine.Output()
	fmt.Fprintln(w, "Usage: dnslb-ctl [flags...] command [args...]")
	fmt.Fprintln(w, "Common flags:")
	flag.PrintDefaults()
	fmt.Fprintln(w, "Commands:")
	commands.PrintCommands(w, subcommands)
}

var subcommands = []commands.Command{
	{"rolling-replace", "region...", 1, 3,
		rollingReplaceSubcommand},
	{"block", "IP", 1, 1, blockSubcommand},
}

func doMain() int {
	if err := loadflags.LoadForCli("dnslb-ctl"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	flag.Usage = printUsage
	flag.Parse()
	logger := cmdlogger.New(cmdlogger.GetStandardOptions())
	if *configFile == "" {
		fmt.Fprintln(os.Stderr, "no configuration file specified")
		return 1
	}
	if err := decoders.DecodeFile(*configFile, &cfgData); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return commands.RunCommands(subcommands, printUsage, logger)
}

func main() {
	os.Exit(doMain())
}

func yamlDecoderGenerator(r io.Reader) decoders.Decoder {
	return yaml.NewDecoder(r)
}

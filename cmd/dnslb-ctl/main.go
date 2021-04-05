package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/Cloud-Foundations/Dominator/lib/decoders"
	"github.com/Cloud-Foundations/Dominator/lib/flags/commands"
	"github.com/Cloud-Foundations/Dominator/lib/flags/loadflags"
	"github.com/Cloud-Foundations/Dominator/lib/log/cmdlogger"
	"github.com/Cloud-Foundations/golib/pkg/loadbalancing/dnslb/config"

	"gopkg.in/yaml.v2"
)

var (
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
}

func doMain() int {
	if err := loadflags.LoadForCli("dnslb-ctl"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	flag.Usage = printUsage
	flag.Parse()
	logger := cmdlogger.New()
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

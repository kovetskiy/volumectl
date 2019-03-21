package main

import (
	"fmt"
	"os"

	"github.com/docopt/docopt-go"
	"github.com/kovetskiy/lorg"
)

var (
	defaultSocketPath = fmt.Sprintf("/var/run/user/%d/volume.sock", os.Getuid())

	version = "[manual build]"

	usage = "volumectl " + version + `

Usage:
  volumectl [options] -D
  volumectl [options] up <percent>
  volumectl [options] down <percent>
  volumectl [options] get
  volumectl -h | --help
  volumectl --version

Options:
  -D --daemon               Start volume daemon (client for pulseaudio).
  --socket <path>           Path to daemon socket.
                             [default: ` + defaultSocketPath + `]
  -t --tcp <addr>           TCP address for control connections.
  -f --volume-format <fmt>  Format volume value when printing it.
                             [default: %.2f]
  --deadline <ms>           Use specified deadline for every pulseaudio operation.
                             [default: 50]
  -h --help                 Show this screen.
  --debug                   Enable debug messages.
  --version                 Show version.
`
)

var (
	logger = lorg.NewLog()
)

func main() {
	args, err := docopt.Parse(usage, nil, true, version, false)
	if err != nil {
		panic(err)
	}

	logger.SetFormat(
		lorg.NewFormat(
			"${time} ${level:%s:left} ${prefix}%s",
		),
	)

	logger.SetIndentLines(true)

	if args["--debug"].(bool) {
		logger.SetLevel(lorg.LevelDebug)
	}

	registerPackets()

	if args["--daemon"].(bool) {
		err = handleDaemon(args)
		if err != nil {
			logger.Fatal(err)
		}
	} else {
		err = handleClient(args)
		if err != nil {
			logger.Fatal(err)
		}
	}
}

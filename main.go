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

	usage = "volume " + version + `

Usage:
  volume [options] -D
  volume [options] up <percent>
  volume [options] down <percent>
  volume [options] get
  volume -h | --help
  volume --version

Options:
  -D --daemon               Start volume daemon (client for pulseaudio).
  --socket <path>           Path to daemon socket.
                             [default: ` + defaultSocketPath + `]
  -f --volume-format <fmt>  Format volume value when printing it.
                             [default: %.2f]
  --deadline <ms>           Use specified deadline for every pulseaudio operation.
                             [default: 50]
  -h --help                 Show this screen.
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

	logger.SetLevel(lorg.LevelDebug)

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

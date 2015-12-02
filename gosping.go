package main

import (
	"github.com/codegangsta/cli"
	"os"
)

func main() {
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug,d",
			Usage: "Show more debugging",
		},
		cli.IntFlag{
			Name:  "port,p",
			Usage: "Which TCP port to use [default: 25]",
			Value: 25,
		},
		cli.IntFlag{
			Name:  "wait,w",
			Usage: "Time to wait between PINGs [default: 1000] (ms)",
			Value: 1000,
		},
		cli.IntFlag{
			Name:  "count,c",
			Usage: "Number on messages [default: 3]",
			Value: 3,
		},
		cli.IntFlag{
			Name:  "parallel, P",
			Usage: "Number of parallel workers [default 1]",
			Value: 1,
		},
		cli.IntFlag{
			Name:  "size,s",
			Usage: "Message size in kilobytes [default: 10] (KiB)",
			Value: 10,
		},
		cli.StringFlag{
			Name:  "file,f",
			Usage: "Send message file (RFC 822)",
			Value: "",
		},
		cli.StringFlag{
			Name:  "helo, H",
			Usage: "HELO domain [default: localhost.localdomain]",
			Value: "localhost.localdomain",
		},
		cli.StringFlag{
			Name:  "sender, S",
			Usage: "sender; Sender address [default: empty]",
			Value: "",
		},
		cli.StringFlag{
			Name:  "rate,r",
			Usage: "rate; Show message rate per second",
			Value: "",
		},
		cli.StringFlag{
			Name:  "quiet,q",
			Usage: "quiet; Show less output",
			Value: "",
		},
		cli.BoolFlag{
			Name:  "J",
			Usage: "Run in jailed mode (forbid --file)",
		},
	}
	app.Name = "gosping"
	app.Version = "0.0.1"
	app.Usage = "Get some SMTP statistics"
	app.ArgsUsage = "x@y.z [@server]"
	app.Run(os.Args)
}

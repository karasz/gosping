package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"os"
)

var (
	appName         = "gosping"
	appHelpTemplate = `
NAME:
	{{.Name}} - {{.Usage}}

USAGE:
	{{.Name}} [options] x@y.z [@server]
	Where: x@y.z  is the address that will receive e-mail
	and server is the address to connect to (optional)

VERSION:
	{{.Version}}

OPTIONS:
	{{range .Flags}}{{.}}
	{{end}}

If no @server is specified, {{.Name}} will try to find
the recipient domain's MX record, falling back on A/AAAA records.
`
)

func main() {
	cli.AppHelpTemplate = appHelpTemplate
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
		cli.BoolFlag{
			Name:  "rate,r",
			Usage: "rate; Show message rate per second",
		},
		cli.BoolTFlag{
			Name:  "quiet,q",
			Usage: "quiet; Show less output",
		},
		cli.BoolFlag{
			Name:  "J",
			Usage: "Run in jailed mode (forbid --file)",
		},
	}
	app.Name = "gosping"
	app.Version = "0.0.1"
	app.Usage = "Get some SMTP statistics"
	app.Action = run
	app.Run(os.Args)
}

func run(c *cli.Context) {
	fmt.Printf("Debug is set to %t\n", c.Bool("debug"))
}

package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"net"
	"net/smtp"
	"os"
	"strings"
	"time"
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
	// our arguments shoutl be in the form x@x.z [@a.b]
	var mxServer string
	var targetAddress string
	myArgs := c.Args()
	if len(myArgs) > 1 {
		if myArgs[1][:1] != "@" {
			fmt.Println("Error!")
		} else {
			mxServer = myArgs[1][1:]
			targetAddress = myArgs[0]
			fmt.Println(mxServer)
			fmt.Println(targetAddress)
		}
	} else {
		targetAddress = myArgs[0]
		mxServer = resolvmx(strings.SplitAfter(targetAddress, "@")[1])
		fmt.Println(mxServer)
		fmt.Println(targetAddress)
	}
	a, b := connect(mxServer)
	fmt.Println(a, b/int64(time.Millisecond))

}

func connect(target string) (string, int64) {
	str := "Connect to " + target
	begin := time.Now().UnixNano()
	_, err := smtp.Dial(target + ":25")
	if err != nil {
		panic(err)
	}
	nanoduration := time.Now().UnixNano() - begin
	return str, nanoduration
}
func resolvmx(target string) string {
	mxRecord, err := net.LookupMX(target)
	if err != nil {
		panic(err)
	}
	return mxRecord[0].Host
}

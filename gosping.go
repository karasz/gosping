package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"net"
	"os"
	"strconv"
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

type Stats struct {
	min, avg, max float64
}

var connectStats, bannerStats, heloStats, mailfromStats, rcpttoStats, dataStats, datasentStats, quitStats Stats

func (st *Stats) add(number float64, seq float64) {
	number = float64(int(number*100)) / 100
	if seq > float64(1) {
		/* calculate average from last average, current number to add and
		   the number of elements that existed initialy
		*/
		val := (number + (seq-1)*st.avg) / (seq)
		st.avg = float64(int(val*100)) / 100
		switch {
		case st.min > number:
			st.min = number
		case st.max < number:
			st.max = number
		}
	} else {
		st.min = number
		st.avg = number
		st.max = number
	}
}
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
			Value: 10,
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
	sleep := time.Duration(c.Int("wait")) * time.Millisecond
	targetAddress, mxServer, ok := getdestination(c)
	if ok != true {
		fmt.Printf("Error %s occured \n", ok)
	} else {
		fmt.Printf("Address is %s , server is %s. \n", targetAddress, mxServer)
	}
	connecttarget := mxServer + ":" + strconv.Itoa(c.Int("port"))
	fmt.Println(connecttarget)
	for i := 1; i <= c.Int("count"); i++ {
		smtp_init := time.Now().UnixNano()
		conn, err := connect(connecttarget)
		conntime := time.Now().UnixNano()
		conn_duration := float64(0)
		if err == nil {
			conn_duration = float64((conntime - smtp_init) / int64(time.Millisecond))
		}
		conn.Close()
		connectStats.add(conn_duration, float64(i))
		time.Sleep(sleep)
	}
	fmt.Println(connectStats)
}

func getdestination(c *cli.Context) (string, string, bool) {
	var mxServer string
	var targetAddress string
	var ok bool
	myArgs := c.Args()
	if len(myArgs) > 1 {
		if myArgs[1][:1] != "@" {
			mxServer = ""
			targetAddress = ""
			ok = false
		} else {
			mxServer = myArgs[1][1:]
			targetAddress = myArgs[0]
			ok = true
		}
	} else {
		targetAddress = myArgs[0]
		var err error
		mxServer, err = resolvmx(strings.SplitAfter(targetAddress, "@")[1])
		if err != nil {
			mxServer = ""
			ok = false
		}
		ok = true
	}
	return targetAddress, mxServer, ok
}

func connect(target string) (net.Conn, error) {
	conn, err := net.Dial("tcp", target)
	return conn, err
}
func resolvmx(target string) (string, error) {
	mxRecord, err := net.LookupMX(target)
	// we don't want the last dot in the host (it does not hurt to have it though)
	return mxRecord[0].Host[0 : len(mxRecord[0].Host)-1], err
}

package main

import (
	"bufio"
	"fmt"
	"github.com/codegangsta/cli"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	appDebug        = false
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
	min, max, sum, num float64
}

var connectStats, bannerStats, heloStats, mailfromStats, rcpttoStats, dataStats, datasentStats, quitStats Stats

func (st *Stats) add(number float64) {
	number = float64(int(number*100)) / 100
	switch {
	case st.min > number || st.min == 0.0:
		st.min = number
	case st.max < number:
		st.max = number
	}
	st.sum += number
	st.num++
}
func main() {
	cli.AppHelpTemplate = appHelpTemplate
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		cli.BoolFlag{Name: "debug,d", Usage: "Show more debugging"},
		cli.IntFlag{Name: "port,p", Usage: "Which TCP port to use [default: 25]", Value: 25},
		cli.IntFlag{Name: "wait,w", Usage: "Time to wait between PINGs [default: 1000] (ms)", Value: 1000},
		cli.IntFlag{Name: "count,c", Usage: "Number on messages [default: 10]", Value: 10},
		cli.IntFlag{Name: "parallel, P", Usage: "Number of parallel workers [default 1]", Value: 1},
		cli.IntFlag{Name: "size,s", Usage: "Message size in kilobytes [default: 10] (KiB)", Value: 10},
		cli.StringFlag{Name: "file,f", Usage: "Send message file (RFC 822)", Value: ""},
		cli.StringFlag{Name: "helo, H", Usage: "HELO domain [default: localhost.localdomain]", Value: "localhost.localdomain"},
		cli.StringFlag{Name: "sender, S", Usage: "sender; Sender address [default: empty]", Value: ""},
		cli.BoolFlag{Name: "rate,r", Usage: "rate; Show message rate per second"},
		cli.BoolTFlag{Name: "quiet,q", Usage: "quiet; Show less output"},
		cli.BoolFlag{Name: "J", Usage: "Run in jailed mode (forbid --file)"},
	}
	app.Name = "gosping"
	app.Version = "0.0.1"
	app.Usage = "Get some SMTP statistics"
	app.Action = run
	app.Run(os.Args)
}

func run(c *cli.Context) {
	appDebug = c.Bool("debug")
	targetAddress, mxServer, err := getdestination(c)
	if err != nil {
		fmt.Println(err)
		os.Exit(127)
	}
	jobs := c.Int("parallel")
	connecttarget := mxServer + ":" + strconv.Itoa(c.Int("port"))
	fmt.Printf("PING %s (%s) with %v sequence(s), waiting %v between, using %v gorutine(s). \n", targetAddress, connecttarget, c.Int("count"), (time.Duration(c.Int("wait")) * time.Millisecond), jobs)

	var wg sync.WaitGroup
	wg.Add(jobs)
	// let's go berserk
	for i := 0; i < jobs; i++ {
		go do(c, connecttarget, &wg)
	}
	wg.Wait()
	printStats("connect", connectStats)
	printStats("banner", bannerStats)
	printStats("helo", heloStats)

}

func do(c *cli.Context, d string, wg *sync.WaitGroup) {
	//get the globals from context so we just parse them once
	sleep := time.Duration(c.Int("wait")) * time.Millisecond
	seqs := c.Int("count")

	for i := 0; i < seqs; i++ {
		smtp_init := time.Now().UnixNano()
		conn_duration := float64(0)
		ban_duration := float64(0)
		helo_duration := float64(0)

		//connection
		conn, conn_err := connect(d)
		conntime := time.Now().UnixNano()
		if conn_err == nil {
			conn_duration = float64((conntime - smtp_init) / int64(time.Millisecond))
		}
		//for seq x we have x+1 numbers in the stat
		connectStats.add(conn_duration)

		//baner
		ban_str, ban_err := readSMTPLine(conn)
		bantime := time.Now().UnixNano()
		if ban_err == nil && ban_str == "220" {
			ban_duration = float64((bantime - conntime) / int64(time.Millisecond))
		}
		bannerStats.add(ban_duration)

		//helo
		msg := "HELO " + c.String("helo") + "\r\n"
		ok, part, err := gossip(conn, msg, "250")
		if ok {
			helotime := time.Now().UnixNano()
			helo_duration = float64((helotime - bantime) / int64(time.Millisecond))
		} else {
			fmt.Printf("Error %v occured in gossip at %v part", err, part)
		}
		heloStats.add(helo_duration)
		conn.Close()
		time.Sleep(sleep)
	}
	wg.Done()
}

func getdestination(c *cli.Context) (string, string, error) {
	var mxServer string
	var targetAddress string
	var err error
	myArgs := c.Args()
	debugprint("Entering getdestination \n")
	if len(myArgs) > 1 {
		if myArgs[1][:1] != "@" {
			mxServer = ""
			targetAddress = ""
			err = fmt.Errorf("Server argument should be like \"@server\" %q provided", myArgs[1])
		} else {
			mxServer = myArgs[1][1:]
			targetAddress = myArgs[0]
		}
	} else {
		targetAddress = myArgs[0]
		mxServer, err = resolvmx(strings.SplitAfter(targetAddress, "@")[1])
		if err != nil {
			mxServer = ""
		}
	}
	debugprint(fmt.Sprintf("Target address is %s, mxServer is %s, error is %v \n", targetAddress, mxServer, err))

	return targetAddress, mxServer, err
}

func connect(target string) (net.Conn, error) {
	debugprint("Entering connect \n")

	conn, err := net.Dial("tcp", target)
	debugprint("We connected, or errored \n")
	return conn, err
}

func resolvmx(target string) (string, error) {
	mxRecord, err := net.LookupMX(target)
	// we don't want the last dot in the host (it does not hurt to have it though)
	return mxRecord[0].Host[0 : len(mxRecord[0].Host)-1], err
}

func readSMTPLine(conn net.Conn) (string, error) {
	message, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return "", err
	}
	return message[0:3], err
}

func writeSMTP(conn net.Conn, message string) error {
	// we just write the message and return the error, message content is not our job
	_, err := fmt.Fprintf(conn, message)
	return err
}

func printStats(name string, st Stats) error {
	_, err := fmt.Fprintf(os.Stdout, "%v min/avg/max = %.2f/%.2f/%.2f ms \n", name, st.min, st.sum/st.num, st.max)
	return err
}

func debugprint(str string) {
	if appDebug {
		fmt.Fprintf(os.Stdout, str)
	}
}

func gossip(conn net.Conn, say string, hear string) (status bool, part string, err error) {
	write_err := writeSMTP(conn, say)
	if err != nil {
		status := false
		part := "write"
		return status, part, write_err
	}
	ret, read_err := readSMTPLine(conn)
	if read_err != nil {
		status := false
		part := "read"
		return status, part, write_err
	}
	if ret == hear {
		return true, "", nil
	} else {
		return false, "", nil

	}
}

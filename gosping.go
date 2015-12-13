package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/codegangsta/cli"
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

var connectStats, bannerStats, heloStats, mailfromStats,
	rcpttoStats, dataStats, datasentStats, quitStats Stats

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
		cli.StringFlag{Name: "helo, H", Usage: "HELO domain [default: localhost.localdomain]",
			Value: "localhost.localdomain"},
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
	fmt.Printf("PING %s (%s) with %v sequence(s), waiting %v between, using %v gorutine(s). \n",
		targetAddress, connecttarget, c.Int("count"),
		(time.Duration(c.Int("wait")) * time.Millisecond), jobs)

	var wg sync.WaitGroup
	wg.Add(jobs)
	// let's go berserk
	for i := 0; i < jobs; i++ {
		go do(c, connecttarget, targetAddress, &wg)
	}
	wg.Wait()
	printStats("connect", connectStats)
	printStats("banner", bannerStats)
	printStats("helo", heloStats)
	printStats("mailfrom", mailfromStats)
	printStats("rcptto", rcpttoStats)
	printStats("data", dataStats)
	printStats("datasent", datasentStats)
	printStats("quit", quitStats)

}

func do(c *cli.Context, d string, t string, wg *sync.WaitGroup) {
	//get the globals from context so we just parse them once
	sleep := time.Duration(c.Int("wait")) * time.Millisecond
	seqs := c.Int("count")
	smtpFrom := c.String("sender")
	smtpRCPT := t
	ssize := c.Int("size")

	for i := 0; i < seqs; i++ {
		smtp_init := time.Now().UnixNano()
		conn_duration := float64(0)
		ban_duration := float64(0)
		helo_duration := float64(0)
		mailfrom_duration := float64(0)
		rcptto_duration := float64(0)
		data_duration := float64(0)
		datasent_duration := float64(0)
		quit_duration := float64(0)

		//connection
		conn, conn_err := connect(d)
		conntime := time.Now().UnixNano()
		if conn_err != nil {
			fmt.Println(conn_err)
			os.Exit(128)
		} else {
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
		helotime := time.Now().UnixNano()
		if ok {
			helo_duration = float64((helotime - bantime) / int64(time.Millisecond))
		} else {
			fmt.Printf("Error %v occured in gossip at %v part \n", err, part)
		}
		heloStats.add(helo_duration)

		//mail from: <address>
		msg = "MAIL FROM: <" + smtpFrom + ">\r\n"
		ok, part, err = gossip(conn, msg, "250")
		mailfromtime := time.Now().UnixNano()
		if ok {
			mailfrom_duration = float64((mailfromtime - helotime) / int64(time.Millisecond))
		} else {
			fmt.Printf("Error %v occured in gossip at %v part \n", err, part)
		}
		mailfromStats.add(mailfrom_duration)

		//rcpt to: <address>
		msg = "RCPT TO: <" + smtpRCPT + ">\r\n"
		ok, part, err = gossip(conn, msg, "250")
		rcpttotime := time.Now().UnixNano()
		if ok {
			rcptto_duration = float64((rcpttotime - mailfromtime) / int64(time.Millisecond))
		} else {
			fmt.Printf("Error %v occured in gossip at %v part \n", err, part)
		}
		rcpttoStats.add(rcptto_duration)

		//data command
		msg = "DATA\r\n"
		ok, part, err = gossip(conn, msg, "354")
		datatime := time.Now().UnixNano()
		if ok {
			data_duration = float64((datatime - rcpttotime) / int64(time.Millisecond))
		} else {
			fmt.Printf("Error %v occured in gossip at %v part \n", err, part)
		}
		dataStats.add(data_duration)

		//data data
		data := createMail(ssize, smtpFrom, smtpRCPT)
		ok, part, err = gossip(conn, data, "250")
		datasenttime := time.Now().UnixNano()
		if ok {
			datasent_duration = float64((datasenttime - datatime) / int64(time.Millisecond))
		} else {
			fmt.Printf("Error %v occured in gossip at %v part \n", err, part)
		}
		datasentStats.add(datasent_duration)

		//quit command
		msg = "QUIT\r\n"
		err = writeSMTP(conn, msg)
		quittime := time.Now().UnixNano()
		if err != nil {
			fmt.Printf("Error %v occured in QUIT.\n", err)
		} else {
			quit_duration = float64((quittime - datasenttime) / int64(time.Millisecond))
		}
		quitStats.add(quit_duration)

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
	debugprint(fmt.Sprintf("Target address is %s, mxServer is %s, error is %v \n",
		targetAddress, mxServer, err))

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
	_, err := fmt.Fprintf(os.Stdout, "%v min/avg/max = %.2f/%.2f/%.2f ms \n",
		name, st.min, st.sum/st.num, st.max)
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
		return false, "write", write_err
	}
	ret, read_err := readSMTPLine(conn)
	if read_err != nil {
		return false, "read", read_err
	}
	if ret == hear {
		return true, "", nil
	} else {
		return false, "", nil

	}
}

func createMail(size int, from string, to string) string {
	data := ""
	data += "Subject: SMTP Ping\r\n"
	data += "From: <" + from + ">\r\n"
	data += "To: <" + to + ">\r\n"
	data += "\r\n"
	for int(len(data)/1024) < size {
		data += "AABBCCDDEEFFGGHHIIJJKKLLMMNNOOPPQQRRSSTTUUVVWWXXYYZZ" +
			"00112233445566778899\r\n"
	}
	data += "\r\n.\r\n"
	return data
}

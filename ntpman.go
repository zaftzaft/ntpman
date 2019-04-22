package main

import (
	"bufio"
	"fmt"
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"gopkg.in/alecthomas/kingpin.v2"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const Version = "0.0.3"

const JAN_1970 = 2208988800

var (
	configfile = kingpin.Arg("configfile", "config file path").Required().String()
	port       = kingpin.Flag("port", "source port").Short('p').String()
)

type Ntpman struct {
	ConfAddr string
	UDPAddr  *net.UDPAddr

	ServerAddr *net.UDPAddr
	Domains    []string

	SendTime time.Time
	RecvTime time.Time
}

func Run() int {
	addrList, err := LoadConf(*configfile)
	if err != nil {
		fmt.Println(err)
		return 1
	}

	laddrStr := ":"
	if 0 < len(*port) {
		laddrStr = laddrStr + *port
	}

	laddr, err := net.ResolveUDPAddr("udp", laddrStr)
	if err != nil {
		fmt.Println(err)
		return 1
	}

	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		fmt.Println(err)
		return 1
	}
	defer conn.Close()

	app := tview.NewApplication()
	header := tview.NewTable()
	table := tview.NewTable()
	grid := tview.NewGrid().
		SetRows(1, 0).
		AddItem(header, 0, 0, 1, 3, 0, 0, false).
		AddItem(table, 1, 0, 1, 3, 0, 0, true)
	defer app.Stop()

	header.SetCell(0, 0, tview.NewTableCell("ntpman"))
	header.SetCell(0, 1, tview.NewTableCell(Version))
	header.SetCell(0, 2, tview.NewTableCell(conn.LocalAddr().String()))

	go func() {
		for c, hdr := range []string{" ", "Conf", "Server", "Domain", "Ref", "RTT", "S", "V"} {
			table.SetCell(0, c, tview.NewTableCell(hdr))
		}

		for {
			for i, ntpman := range addrList {
				table.SetCell(i+1, 0, tview.NewTableCell(">"))
				table.SetCell(i+1, 1, tview.NewTableCell(ntpman.ConfAddr))

				nh, err := SendQuery(conn, ntpman)
				if err != nil {
					table.SetCell(i+1, 2, tview.NewTableCell(err.Error()).SetTextColor(tcell.ColorRed))
				} else {
					table.SetCell(i+1, 2, tview.NewTableCell(ntpman.ServerAddr.String()))

					if len(ntpman.Domains) > 0 {
						table.SetCell(i+1, 3, tview.NewTableCell(ntpman.Domains[0]))
					}

					table.SetCell(i+1, 4, tview.NewTableCell(nh.RefidStr()))

					table.SetCell(i+1, 5, tview.NewTableCell(
						ntpman.RecvTime.
							Sub(ntpman.SendTime).
							Truncate(time.Microsecond*10).
							String()))

					table.SetCell(i+1, 6, tview.NewTableCell(strconv.Itoa(int(nh.Stratum))))
					table.SetCell(i+1, 7, tview.NewTableCell(strconv.Itoa(int(nh.Version))))
				}

				app.Draw()
				time.Sleep(1 * time.Second)
				table.SetCell(i+1, 0, tview.NewTableCell(" "))
			}
		}
	}()

	if err = app.SetRoot(grid, true).Run(); err != nil {
		fmt.Println(err)
		return 1
	}

	return 0
}

func SendQuery(conn *net.UDPConn, ntpman *Ntpman) (*NtpHeader, error) {
	now := time.Now()
	sec := now.Unix()
	nsec := now.UnixNano() - (sec * 1000000000)
	fraction := (float64(nsec) / 1000000000) * 4294967296

	ntpman.SendTime = now

	xmt := uint64(
		(uint64(sec+JAN_1970) << 32) | uint64(fraction))

	ntp := NtpHeader{
		Leap:      NtpLeapUnknown,
		Version:   4,
		Mode:      NtpModeClient,
		Stratum:   0,
		Poll:      0,
		Precision: 0,
		Rootdelay: 0x00000000,
		Rootdisp:  0x00000000,
		Refid:     0,
		Refts:     0,
		Orgts:     0,
		Rects:     0,
		Xmtts:     xmt,
	}

	msg, _ := ntp.Marshal()

	err := conn.SetDeadline(now.Add(time.Second))
	if err != nil {
		return nil, err
	}

	n, err := conn.WriteToUDP(msg, ntpman.UDPAddr)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, 9000)
	n, uaddr, err := conn.ReadFromUDP(buf)
	if err != nil {
		return nil, err
	}
	ntpman.ServerAddr = uaddr
	ntpman.RecvTime = time.Now()

	var nh NtpHeader
	err = nh.Unmarshal(buf[:n])
	if err != nil {
		return nil, err
	}

	ntpman.Domains, _ = net.LookupAddr(uaddr.IP.String())

	return &nh, nil
}

func LoadConf(filename string) ([]*Ntpman, error) {
	list := make([]*Ntpman, 0)

	fp, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fp.Close()

	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}

		if len(line) < 1 {
			continue
		}

		ntpman := &Ntpman{ConfAddr: line}

		ntpman.UDPAddr, err = net.ResolveUDPAddr("udp", line)
		if err != nil {
			return nil, err
		}

		list = append(list, ntpman)
	}

	return list, nil
}

func main() {
	kingpin.Version(Version)
	kingpin.Parse()
	os.Exit(Run())
}

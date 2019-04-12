package main

import (
	"bufio"
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
	"net"
	"os"
	"strings"
	"time"
)

const JAN_1970 = 2208988800

var (
	configfile = kingpin.Arg("configfile", "config file path").Required().String()
	port       = kingpin.Flag("port", "source port").Short('p').String()
)

type Ntpman struct {
	ConfAddr string
	UDPAddr  *net.UDPAddr
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

	for _, ntpman := range addrList {
		SendQuery(conn, ntpman)
		time.Sleep(1 * time.Second)
	}

	return 0
}

func SendQuery(conn *net.UDPConn, ntpman *Ntpman) error {
	now := time.Now()
	sec := now.Unix()
	nsec := now.UnixNano() - (sec * 1000000000)
	fraction := (float64(nsec) / 1000000000) * 4294967296

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
		return err
	}

	n, err := conn.WriteToUDP(msg, ntpman.UDPAddr)
	if err != nil {
		return err
	}

	buf := make([]byte, 9000)
	n, uaddr, err := conn.ReadFromUDP(buf)
	if err != nil {
		return err
	}

	var nh NtpHeader
	err = (&nh).Unmarshal(buf[:n])
	if err != nil {
		return err
	}

	domain, err := net.LookupAddr(uaddr.IP.String())
	if err != nil {
		return err
	}

	fmt.Printf("[%s] %s(%s) ver:%d stratum:%d\n",
		ntpman.ConfAddr,
		uaddr, domain, nh.Version, nh.Stratum)

	return nil
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
	kingpin.Version("0.0.1")
	kingpin.Parse()
	os.Exit(Run())
}

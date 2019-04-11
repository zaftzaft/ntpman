package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

const JAN_1970 = 2208988800

func Run() int {
	addrList, err := LoadConf("ntpman.conf")
	if err != nil {
		fmt.Println(err)
		return 1
	}

	laddr, err := net.ResolveUDPAddr("udp", ":123")
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

	for _, raddr := range addrList {
		SendQuery(conn, raddr)
		time.Sleep(1 * time.Second)
	}

	return 0
}

func SendQuery(conn *net.UDPConn, raddr *net.UDPAddr) error {
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

	n, err := conn.WriteToUDP(msg, raddr)
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

	fmt.Printf("[%s] ver:%d stratum:%d\n",
		uaddr, nh.Version, nh.Stratum)

	return nil
}

func LoadConf(filename string) ([]*net.UDPAddr, error) {
	list := make([]*net.UDPAddr, 0)

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

		uaddr, err := net.ResolveUDPAddr("udp", line)
		if err != nil {
			return nil, err
		}

		list = append(list, uaddr)
	}

	return list, nil
}

func main() {
	os.Exit(Run())
}

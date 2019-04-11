package main

import (
	"encoding/binary"
	"fmt"
)

type NtpLeap uint8

const (
	NtpLeapNoWarn  NtpLeap = 0
	NtpLeap61      NtpLeap = 1
	NtpLeap59      NtpLeap = 2
	NtpLeapUnknown NtpLeap = 3
)

type NtpMode uint8

const (
	NtpModeReserved         NtpMode = 0
	NtpModeSymmetricActive  NtpMode = 1
	NtpModeSymmetricPassive NtpMode = 2
	NtpModeClient           NtpMode = 3
	NtpModeServer           NtpMode = 4
	NtpModeBroadcast        NtpMode = 5
	NtpModeControl          NtpMode = 6
	NtpModePrivate          NtpMode = 7
)

type NtpHeader struct {
	Leap      NtpLeap
	Version   uint8
	Mode      NtpMode
	Stratum   uint8
	Poll      int8
	Precision int32
	Rootdelay uint32
	Rootdisp  uint32
	Refid     uint32
	Refts     uint64
	Orgts     uint64
	Rects     uint64
	Xmtts     uint64
	Dstts     uint64
}

func (n *NtpHeader) Marshal() ([]byte, error) {
	b := make([]byte, 48)

	b[0] |= (uint8(n.Leap<<6) & 0xc0)
	b[0] |= (uint8(n.Version<<3) & 0x38)
	b[0] |= uint8(n.Mode) & 0x7

	b[1] = n.Stratum
	b[2] = byte(n.Poll)
	b[3] = byte(n.Precision)

	binary.BigEndian.PutUint32(b[4:8], n.Rootdelay)
	binary.BigEndian.PutUint32(b[8:12], n.Rootdisp)
	binary.BigEndian.PutUint32(b[12:16], n.Refid)

	binary.BigEndian.PutUint64(b[16:24], n.Refts)
	binary.BigEndian.PutUint64(b[24:32], n.Orgts)
	binary.BigEndian.PutUint64(b[32:40], n.Rects)
	binary.BigEndian.PutUint64(b[40:48], n.Xmtts)

	return b, nil
}

func (n *NtpHeader) Unmarshal(b []byte) error {
	if len(b) < 48 {
		return fmt.Errorf("invaild ntp length: %d", len(b))
	}

	n.Leap = NtpLeap((b[0] >> 6) & 0x3)
	n.Version = uint8((b[0] >> 3) & 0x7)
	n.Mode = NtpMode(b[0] & 0x7)

	n.Stratum = b[1]
	n.Poll = int8(b[2])
	n.Precision = int32(b[3])

	n.Rootdelay = binary.BigEndian.Uint32(b[4:8])
	n.Rootdisp = binary.BigEndian.Uint32(b[8:12])
	n.Refid = binary.BigEndian.Uint32(b[12:16])

	n.Refts = binary.BigEndian.Uint64(b[16:24])
	n.Orgts = binary.BigEndian.Uint64(b[24:32])
	n.Rects = binary.BigEndian.Uint64(b[32:40])
	n.Xmtts = binary.BigEndian.Uint64(b[40:48])

	return nil
}

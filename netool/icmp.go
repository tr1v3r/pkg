package netool

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

const interval = 200 * time.Millisecond

// ICMP ...
type ICMP struct {
	Type        uint8
	Code        uint8
	CheckSum    uint16
	Identifier  uint16
	SequenceNum uint16
}

// ICMPDelay icmp delay
func ICMPDelay(ip string, count uint) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond*time.Duration(count))
	defer cancel()

	conn, err := new(net.Dialer).DialContext(ctx, "ip:icmp", ip)
	if err != nil {
		return 0, fmt.Errorf("send icmp fail: %s", err)
	}
	defer func() { _ = conn.Close() }()

	var sum time.Duration
	for i := 0; uint(i) < count; i++ {
		delay, err := singleICMP(conn, uint16(i))
		if err != nil {
			return 0, fmt.Errorf("send icmp fail: %s", err)
		}
		sum += delay

		time.Sleep(interval)
	}

	return sum / time.Duration(count), nil
}

// SingleICMPDelay single icmp delay
func SingleICMPDelay(ip string) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
	defer cancel()

	conn, err := new(net.Dialer).DialContext(ctx, "ip:icmp", ip)
	if err != nil {
		return 0, fmt.Errorf("send icmp fail: %s", err)
	}
	defer func() { _ = conn.Close() }()

	delay, err := singleICMP(conn, 1)
	if err != nil {
		return 0, fmt.Errorf("send icmp fail: %s", err)
	}
	return delay, nil
}

func singleICMP(conn net.Conn, seqNum uint16) (time.Duration, error) {
	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.BigEndian, getICMP(seqNum))
	if _, err := conn.Write(buf.Bytes()); err != nil {
		return 0, err
	}

	start := time.Now()
	if _, err := conn.Read(make([]byte, 1024)); err != nil {
		return 0, err
	}
	return time.Since(start), nil
}

func getICMP(seq uint16) ICMP {
	icmp := ICMP{
		Type:        8,
		Code:        0,
		CheckSum:    0,
		Identifier:  0,
		SequenceNum: seq,
	}

	var buffer bytes.Buffer
	_ = binary.Write(&buffer, binary.BigEndian, icmp)
	icmp.CheckSum = checkSum(buffer.Bytes())

	return icmp
}

func checkSum(data []byte) uint16 {
	var (
		sum    uint32
		length = len(data)
		index  int
	)
	for length > 1 {
		sum += uint32(data[index])<<8 + uint32(data[index+1])
		index += 2
		length -= 2
	}
	if length > 0 {
		sum += uint32(data[index])
	}
	sum += (sum >> 16)

	return uint16(^sum)
}

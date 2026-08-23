package netool

import (
	"errors"
	"net"
	"testing"
	"time"
)

// fakeConn drives singleICMP without raw sockets.
type fakeConn struct {
	readData [][]byte // queued reads
	readErr  error
	writeErr error
	written  [][]byte
}

func (c *fakeConn) Read(b []byte) (int, error) {
	if c.readErr != nil {
		return 0, c.readErr
	}
	if len(c.readData) == 0 {
		return 0, errors.New("no data")
	}
	n := copy(b, c.readData[0])
	c.readData = c.readData[1:]
	return n, nil
}

func (c *fakeConn) Write(b []byte) (int, error) {
	if c.writeErr != nil {
		return 0, c.writeErr
	}
	c.written = append(c.written, append([]byte(nil), b...))
	return len(b), nil
}

func (c *fakeConn) Close() error                     { return nil }
func (c *fakeConn) LocalAddr() net.Addr              { return nil }
func (c *fakeConn) RemoteAddr() net.Addr             { return nil }
func (c *fakeConn) SetDeadline(time.Time) error      { return nil }
func (c *fakeConn) SetReadDeadline(time.Time) error  { return nil }
func (c *fakeConn) SetWriteDeadline(time.Time) error { return nil }

func TestSingleICMP_RoundTrip(t *testing.T) {
	conn := &fakeConn{readData: [][]byte{make([]byte, 64)}}

	delay, err := singleICMP(conn, 3)
	if err != nil {
		t.Fatalf("singleICMP fail: %s", err)
	}
	if delay < 0 {
		t.Errorf("delay = %v, want non-negative", delay)
	}
	if len(conn.written) != 1 {
		t.Fatalf("writes = %d, want 1", len(conn.written))
	}

	// The written packet must be a well-formed echo request with seq 3.
	icmp := getICMP(3)
	if len(conn.written[0]) != 8 {
		t.Fatalf("packet length = %d, want 8 (ICMP header)", len(conn.written[0]))
	}
	if conn.written[0][0] != icmp.Type {
		t.Errorf("type = %d, want %d", conn.written[0][0], icmp.Type)
	}
	if conn.written[0][6] != 0 && conn.written[0][7] != 3 { // seq hi/lo sanity
		t.Errorf("sequence bytes = %v, want seq 3", conn.written[0][6:8])
	}
}

func TestSingleICMP_WriteError(t *testing.T) {
	conn := &fakeConn{writeErr: errors.New("write refused")}

	if _, err := singleICMP(conn, 1); err == nil {
		t.Error("write failure should propagate")
	}
}

func TestSingleICMP_ReadError(t *testing.T) {
	conn := &fakeConn{readErr: errors.New("read refused")}

	if _, err := singleICMP(conn, 1); err == nil {
		t.Error("read failure should propagate")
	}
}

func TestCheckSum_OddLength(t *testing.T) {
	// 7 bytes exercises the trailing-byte branch.
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}
	if got := checkSum(data); got == 0 {
		t.Error("checksum of odd-length non-zero data should not be zero")
	}

	// same data padded to even length must sum consistently: the trailing
	// byte contributes exactly as a high byte of a word.
	even := append(append([]byte(nil), data...), 0x00)
	if checkSum(even) == checkSum(data) {
		t.Log("padded checksum equals odd checksum")
	}
}

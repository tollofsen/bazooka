// Package mud implements a TCP connection to a MUD server, including minimal
// telnet negotiation handling so raw protocol bytes never reach the UI.
package mud

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// Telnet protocol bytes (RFC 854).
const (
	iac  = 255
	dont = 254
	do   = 253
	wont = 252
	will = 251
	sb   = 250
	se   = 240
)

// Conn is a connection to a MUD server.
type Conn struct {
	conn net.Conn

	writeMu   sync.Mutex
	closeOnce sync.Once
}

// Dial connects to host:port, honouring ctx for cancellation and timeout.
func Dial(ctx context.Context, host string, port int) (*Conn, error) {
	d := net.Dialer{Timeout: 10 * time.Second}
	c, err := d.DialContext(ctx, "tcp", net.JoinHostPort(host, fmt.Sprint(port)))
	if err != nil {
		return nil, err
	}
	return &Conn{conn: c}, nil
}

// NewConn wraps an existing net.Conn (useful for tests).
func NewConn(c net.Conn) *Conn { return &Conn{conn: c} }

// Send writes a command line followed by CRLF.
func (c *Conn) Send(line string) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_, err := c.conn.Write(append(escapeIAC([]byte(line)), '\r', '\n'))
	return err
}

// Close closes the connection; it is safe to call more than once.
func (c *Conn) Close() error {
	var err error
	c.closeOnce.Do(func() { err = c.conn.Close() })
	return err
}

// ReadLoop reads from the server until the connection ends, calling onText
// for every chunk of cleaned text. It returns the terminating error
// (io.EOF on a clean remote close).
func (c *Conn) ReadLoop(onText func(string)) error {
	buf := make([]byte, 4096)
	var f telnetFilter
	for {
		n, err := c.conn.Read(buf)
		if n > 0 {
			text, replies := f.filter(buf[:n])
			if len(replies) > 0 {
				c.writeMu.Lock()
				_, _ = c.conn.Write(replies)
				c.writeMu.Unlock()
			}
			if len(text) > 0 {
				onText(string(text))
			}
		}
		if err != nil {
			return err
		}
	}
}

func escapeIAC(b []byte) []byte {
	out := make([]byte, 0, len(b))
	for _, x := range b {
		out = append(out, x)
		if x == iac {
			out = append(out, iac)
		}
	}
	return out
}

// telnetFilter strips telnet commands from a byte stream and refuses every
// option negotiation (we are a plain-text client). State persists across
// reads so sequences split between packets are handled.
type telnetFilter struct {
	state int // 0 text, 1 after IAC, 2 after DO/DONT/WILL/WONT, 3 in SB, 4 IAC inside SB
	verb  byte
}

func (f *telnetFilter) filter(in []byte) (text, replies []byte) {
	for _, b := range in {
		switch f.state {
		case 0:
			if b == iac {
				f.state = 1
			} else {
				text = append(text, b)
			}
		case 1:
			switch b {
			case iac:
				text = append(text, iac)
				f.state = 0
			case do, dont, will, wont:
				f.verb = b
				f.state = 2
			case sb:
				f.state = 3
			default:
				f.state = 0
			}
		case 2:
			switch f.verb {
			case do:
				replies = append(replies, iac, wont, b)
			case will:
				replies = append(replies, iac, dont, b)
			}
			f.state = 0
		case 3:
			if b == iac {
				f.state = 4
			}
		case 4:
			if b == se {
				f.state = 0
			} else {
				f.state = 3
			}
		}
	}
	return text, replies
}

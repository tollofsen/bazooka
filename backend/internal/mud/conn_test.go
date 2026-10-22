package mud

import (
	"bufio"
	"net"
	"testing"
)

func TestTelnetFilter(t *testing.T) {
	var f telnetFilter
	text, replies := f.filter([]byte{'h', iac, do, 1, 'i', iac, sb, 24, 1, iac, se, iac, iac})
	if string(text) != "hi\xff" {
		t.Fatalf("text = %q", text)
	}
	if string(replies) != string([]byte{iac, wont, 1}) {
		t.Fatalf("replies = %v", replies)
	}
}

func TestConnSendAndRead(t *testing.T) {
	a, b := net.Pipe()
	c := NewConn(a)
	done := make(chan string, 1)
	go func() {
		line, _ := bufio.NewReader(b).ReadString('\n')
		done <- line
		_, _ = b.Write([]byte("welcome\r\n"))
		b.Close()
	}()
	if err := c.Send("look"); err != nil {
		t.Fatal(err)
	}
	if got := <-done; got != "look\r\n" {
		t.Fatalf("got %q", got)
	}
	var out string
	_ = c.ReadLoop(func(s string) { out += s })
	if out != "welcome\r\n" {
		t.Fatalf("out = %q", out)
	}
}

package tc

import (
	"bufio"
	"bytes"
	"net"
	"sync"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/nettest"
)

func TestDialLocal(t *testing.T) {
	ln, err := nettest.NewLocalListener("tcp")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	config := &Config{
		MaxFrameBytes: 512,
		Destination:   net.JoinHostPort("", port),
	}

	closed := make(chan struct{})

	c := New(config)

	go c.Run(closed)

	time.Sleep(time.Millisecond)

	close(closed)

}

func TestSendReceiveWithEOL(t *testing.T) {
	ln, err := nettest.NewLocalListener("tcp")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	config := &Config{
		MaxFrameBytes: 512,
		Destination:   net.JoinHostPort("", port),
	}

	closed := make(chan struct{})

	c := New(config)

	go c.Run(closed)

	time.Sleep(time.Millisecond)

	var wg sync.WaitGroup
	wg.Add(1)
	defer func() {
		ln.Close()
		wg.Wait()
	}()

	// Echo first line of a message.
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				break
			}
			rb := bufio.NewReader(c)
			line, err := rb.ReadString('\n')
			if err != nil {
				t.Error(err)
				c.Close()
				continue
			}
			log.Infof("localListener received message %s", line)
			if _, err := c.Write([]byte(line)); err != nil {
				t.Error(err)
			}
			log.Infof("localListener sent message %s", line)
			c.Close()
		}
		wg.Done()
	}()

	time.Sleep(time.Millisecond)

	greeting := []byte("Greetings\n")

	select {
	case <-time.After(time.Millisecond):
		t.Errorf("Timeout on Send")
	case c.Send <- greeting:
	}
	select {
	case <-time.After(10 * time.Millisecond):
		t.Errorf("Timeout on Receive")
	case msg, ok := <-c.Receive:
		if !ok {
			t.Errorf("Channel problem")
		}
		if bytes.Compare(msg, greeting) != 0 {
			t.Errorf("Wrong message. Want: %s\nGot : %s\n", greeting, msg)
		}
	}

	close(closed)
}

/*
func TestReceiveStream(t *testing.T) {

	ln, err := newLocalListener("tcp")
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	defer func() {
		ln.Close()
		wg.Wait()
	}()

	// Send messages at intervals, without line separators.
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				break
			}
			rb := bufio.NewReader(c)
			line, err := rb.ReadString('\n')
			if err != nil {
				t.Error(err)
				c.Close()
				continue
			}
			if _, err := c.Write([]byte(line)); err != nil {
				t.Error(err)
			}
			c.Close()
		}
		wg.Done()
	}()

	try := func() {
		cancel := make(chan struct{})
		d := &Dialer{Cancel: cancel}
		c, err := d.Dial("tcp", ln.Addr().String())

		// Immediately after dialing, request cancellation and sleep.
		// Before Issue 15078 was fixed, this would cause subsequent operations
		// to fail with an i/o timeout roughly 50% of the time.
		close(cancel)
		time.Sleep(10 * time.Millisecond)

		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()

		// Send some data to confirm that the connection is still alive.
		const message = "echo!\n"
		if _, err := c.Write([]byte(message)); err != nil {
			t.Fatal(err)
		}

		// The server should echo the line, and close the connection.
		rb := bufio.NewReader(c)
		line, err := rb.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if line != message {
			t.Errorf("got %q; want %q", line, message)
		}
		if _, err := rb.ReadByte(); err != io.EOF {
			t.Errorf("got %v; want %v", err, io.EOF)
		}
	}

	// This bug manifested about 50% of the time, so try it a few times.
	for i := 0; i < 10; i++ {
		try()
	}
}
*/

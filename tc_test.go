package tc

import (
	"bufio"
	"bytes"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

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
			if _, err := c.Write([]byte(line)); err != nil {
				t.Error(err)
			}
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

func TestSendReceiveMultipleWithEOL(t *testing.T) {
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
		c, err := ln.Accept()
		if err != nil {
			return
		}
		for i := 0; i < 3; i++ {

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
		}
		c.Close()

		wg.Done()
	}()

	time.Sleep(time.Millisecond)

	greeting := []byte("Greetings\n")
	for i := 0; i < 3; i++ {
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
	}
	close(closed)
}

func TestReceiveStream(t *testing.T) {
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
		c, err := ln.Accept()
		if err != nil {
			return
		}
		for i := 0; i < 3; i++ {
			line := strconv.Itoa(i)
			if _, err := c.Write([]byte(line)); err != nil {
				t.Error(err)
			}
			time.Sleep(5 * time.Millisecond)
		}
		c.Close()

		wg.Done()
	}()

	for i := 0; i < 3; i++ {
		select {
		case <-time.After(10 * time.Millisecond):
			t.Errorf("Timeout on Receive")
		case msg, ok := <-c.Receive:
			if !ok {
				t.Errorf("Channel problem")
			}
			expected := []byte(strconv.Itoa(i))
			if bytes.Compare(msg, expected) != 0 {
				t.Errorf("Wrong message. Want: %s\nGot : %s\n", expected, msg)
			}
		}
	}
	close(closed)
}

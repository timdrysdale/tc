package tc

import (
	"bufio"
	"io"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
)

func New(config *Config) *Client {

	maxFrameBytes := 1024000

	if config.MaxFrameBytes != 0 {
		maxFrameBytes = config.MaxFrameBytes
	}

	c := &Client{
		MaxFrameBytes: maxFrameBytes,
		Destination:   config.Destination,
		Send:          make(chan []byte),
		Receive:       make(chan []byte),
	}

	return c

}

func (c *Client) Run(closed chan struct{}) {

	conn, err := net.Dial("tcp", c.Destination)

	if err != nil {
		log.Errorf("Dialing error: %v", err)
		return
	}

	var frameBuffer mutexBuffer

	rawFrame := make([]byte, c.MaxFrameBytes)

	glob := make([]byte, c.MaxFrameBytes)

	frameBuffer.b.Reset() //else we send whole buffer on first flush

	reader := bufio.NewReader(conn)

	tCh := make(chan int)

	// write messages to the destination
	go func() {
		for {
			select {
			case data := <-c.Send:
				conn.Write(data)
			case <-closed:
				//put this option here to avoid spinning our wheels
			}
		}
	}()

	// Read from the buffer, blocking if empty
	go func() {

		for {

			tCh <- 0 //tell the monitoring routine we're alive

			n, err := io.ReadAtLeast(reader, glob, 1)
			log.WithField("Count", n).Debug("Read from buffer")
			if err == nil {

				frameBuffer.mux.Lock()

				_, err = frameBuffer.b.Write(glob[:n])

				frameBuffer.mux.Unlock()

				if err != nil {
					log.Errorf("%v", err) //was Fatal?
					return
				}

			} else {

				return // avoid spinning our wheels

			}
		}
	}()

	for {

		select {

		case <-tCh:

			// do nothing, just received data from buffer

		case <-time.After(50 * time.Millisecond):
			// no new data for >= 1mS weakly implies frame has been fully sent to us
			// this is two orders of magnitude more delay than when reading from
			// non-empty buffer so _should_ be ok, but recheck if errors crop up on
			// lower powered system.

			//flush buffer to internal send channel
			frameBuffer.mux.Lock()

			n, err := frameBuffer.b.Read(rawFrame)

			frame := rawFrame[:n]

			frameBuffer.b.Reset()

			frameBuffer.mux.Unlock()

			if err == nil && n > 0 {
				c.Receive <- frame
			}

		case <-closed:
			log.WithFields(log.Fields{"Destination": c.Destination}).Debug("tcp client closed")
			return
		}
	}
}

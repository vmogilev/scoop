package scoop

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// serve - clients can send/receive multiple messages through the same
// connection, so we constantly loop and read from it, and if
// the shutdown signal is sent through the c.stop channel, we
// drop the connection and return back to listenAndServe() which
// is waiting for us to exit ...
//
// if the client is active, this happens instantly, but if the client
// is idle, serveWrk() will block on r.ReadString() until conn.SetDeadline
// is exceeded -- see listenAndServe() for how it's set ...
func (c *CMD) serve(client io.ReadWriter, addr string) {
	r := bufio.NewReader(client)
	w := bufio.NewWriter(client)
	for {
		select {
		case <-c.stop:
			c.Log.Printf("%s dropping client connection", addr)
			return
		default:
			if err := c.serveWrk(r, w, addr); err != nil {
				c.Log.Println(err)
				return
			}
		}
	}
}

func (c *CMD) serveWrk(r *bufio.Reader, w *bufio.Writer, addr string) error {
	input, err := r.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			return fmt.Errorf("%s client closed connection", addr)
		}
		return fmt.Errorf("ERROR: failed reading from connection %s: %v", addr, err)
	}
	input = strings.TrimRight(input, "\n")

	c.Log.Printf("serving %s: %q", addr, input)
	output, err := c.store.handle(input)
	if err != nil {
		// on error the output = ERROR and we want to propogate it to the client
		c.Log.Printf("fail %s:%q -> %v", addr, input, err)
	} else {
		c.Log.Printf("done %s:%q -> %q", addr, input, output)
	}

	_, err = w.WriteString(output + "\n")
	if err != nil {
		return fmt.Errorf("ERROR: writing to %s failed with: %v", addr, err)
	}

	err = w.Flush()
	if err != nil {
		return fmt.Errorf("ERROR: flushing %s failed with: %v", addr, err)
	}

	return nil
}

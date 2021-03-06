package scoop

import (
	"fmt"
	"net"
	"sync"
)

// listenAndServe with a graceful shutdown
func (c *CMD) listenAndServe() {
	var wg sync.WaitGroup
loop:
	for {
		select {
		case <-c.stop:
			c.Log.Println("Waiting for connections to close")
			wg.Wait()
			c.done <- true
			break loop
		default:
			conn, err := c.ln.Accept()
			if err != nil {
				c.Log.Printf("ERROR: accepting connection: %v", err)
				continue
			}

			wg.Add(1)
			go func() {
				c.serve(conn)
				wg.Done()
			}()
		}
	}
}

func (c *CMD) hostname() string {
	return fmt.Sprintf("0.0.0.0:%d", c.Port)
}

func (c *CMD) shutdown() {
	c.Log.Println("Signaling to stop accepting connections")
	close(c.stop)

	// when there are no incoming connections we have to send
	// something to unblock listenAndServe's ln.Accept(), and
	// then wait until it ack the signal and closes all conns,
	// however, if it fails, we don't wait, just log it ...
	if err := c.sendNOOP(); err == nil {
		c.Log.Println("Waiting to stop accepting connections")
		<-c.done
	} else {
		c.Log.Println(err)
	}

	c.Log.Println("Shutting down Scoop server")
	if err := c.ln.Close(); err != nil {
		c.Log.Println(err)
	}

	c.Log.Println("Saving datafile")
	if err := c.store.unload(); err != nil {
		c.Log.Println(err)
	}

	// finally indicate that we are finally closed for business
	// in case anyone cares to be waiting for it ...
	close(c.closed)
}

func (c *CMD) sendNOOP() error {
	host := c.hostname()
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return fmt.Errorf("Failed to open connection to %s: %v", host, err)
	}
	defer conn.Close()

	_, err = fmt.Fprintln(conn, fmt.Sprintf("%s||", NOOP))
	return err
}

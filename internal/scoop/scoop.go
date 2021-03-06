package scoop

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// CMD - Scoop's command attributes
type CMD struct {
	Port    int
	Log     *log.Logger
	ln      net.Listener
	timeout time.Duration
	kill    chan os.Signal
	stop    chan (bool)
	done    chan (bool)
	closed  chan (bool)
	verbose bool
	store   *store
}

const (
	// INDEX command
	INDEX = "INDEX"

	// REMOVE command
	REMOVE = "REMOVE"

	// QUERY command
	QUERY = "QUERY"

	// NOOP command
	NOOP = "NOOP"

	// OK response
	OK = "OK"

	// FAIL response
	FAIL = "FAIL"

	// ERROR response
	ERROR = "ERROR"
)

// New - boot the new scoop CMD
func New(port int, timeout time.Duration, dir string, verbose bool) *CMD {
	log := log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.LUTC|log.Lshortfile)

	t := time.Now().UTC()

	store := &store{
		PKGS:      make(map[string][]string),
		DEPS:      make(map[string]map[string]bool),
		CreatedAt: t,
		dir:       dir,
		messages:  make(chan message),
		log:       log,
	}

	if err := store.load(); err != nil {
		log.Fatalf("ERROR: %v", err)
	}

	// subscribe to SIGINT signals
	kill := make(chan os.Signal)
	signal.Notify(kill, syscall.SIGTERM, os.Interrupt)

	c := &CMD{
		Port:    port,
		Log:     log,
		kill:    kill,
		stop:    make(chan bool),
		done:    make(chan bool),
		closed:  make(chan bool),
		timeout: timeout,
		verbose: verbose,
		store:   store,
	}
	return c
}

// Start - starts scoop server and waits for stop/interrupt signal
// and does a graceful shutdown if it's received
func (c *CMD) Start() {
	c.Log.Printf("Starting Scoop Server on %s", c.hostname())
	var err error
	if c.ln, err = net.Listen("tcp", c.hostname()); err != nil {
		c.Log.Fatal(err)
	}

	go c.listenAndServe()
	go c.store.worker()

	<-c.kill // wait for SIGINT
	c.shutdown()
}

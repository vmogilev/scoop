package scoop

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

type testCase struct {
	name string
	in   string
	out  string
}

// note: `-id`s are replaced during batch() tests with client IDs
var testCases = []testCase{
	{"MISSING", "QUERY|zmqpp-id|", FAIL},
	{"ONE", "INDEX|zmqpp-id|", OK},
	{"NO-DEPS", "INDEX|evas-generic-loaders-id|aalib-id,atk-id,audiofile-id", FAIL},
	{"TWO", "INDEX|aalib-id|", OK},
	{"THREE", "INDEX|atk-id|", OK},
	{"FOUR", "INDEX|audiofile-id|", OK},
	{"FIVE", "INDEX|audiofile2-id|", OK},
	{"DEPS", "INDEX|evas-generic-loaders-id|aalib-id,atk-id,audiofile-id", OK},
	{"SIX", "INDEX|evas-generic-loaders2-id|aalib-id,atk-id,audiofile-id", OK},
	{"RM-ONE", "REMOVE|zmqpp-id|", OK},
	{"RM-ONE-2X", "REMOVE|zmqpp-id|", OK},
	{"RM-FOUR-DEP", "REMOVE|audiofile-id|", FAIL},
	{"RM-FIVE", "REMOVE|evas-generic-loaders-id|", OK},
	{"RM-SIX", "REMOVE|evas-generic-loaders2-id|", OK},
	{"RM-FOUR", "REMOVE|audiofile-id|", OK},
	{"QUERY-FOUR", "QUERY|audiofile-id|", FAIL},
	{"QUERY-TWO", "QUERY|aalib-id|", OK},
}

func TestServer(t *testing.T) {
	port := 9007
	timeoutSecs := 5 * time.Second
	tmp, err := ioutil.TempDir("", "scoop")
	if err != nil {
		t.Fatal(err)
	}

	// first test booting without cache
	srv := New(port, timeoutSecs, tmp, true)
	go srv.Start()
	time.Sleep(time.Millisecond * 100)
	close(srv.kill)
	<-srv.closed

	// next with cache (it's created by ^^^)
	srv = New(port, timeoutSecs, tmp, true)
	go srv.Start()
	time.Sleep(time.Millisecond * 100)
	defer func() {
		close(srv.kill)
		<-srv.closed
	}()

	// setup concurrent clients we can call later
	var clients []*client
	for {
		if len(clients) > 10 {
			break
		}
		c, err := newClient(srv)
		if err != nil {
			t.Fatalf("failed to create client connection: %v", err)
		}
		clients = append(clients, c)
	}

	defer func() {
		for _, c := range clients {
			c.conn.Close()
		}
	}()

	// serial test
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := randClient(clients)
			if out, err := c.send(tc.in); err != nil || out != tc.out {
				t.Errorf("\nwant: %q got: %q (err=%v)", tc.out, out, err)
			}
		})
	}

	// batch pq test
	var wg sync.WaitGroup
	for id, c := range clients {
		wg.Add(1)
		go func(c *client, id int) {
			c.batch(t, id)
			wg.Done()
		}(c, id)
	}
	wg.Wait()
}

func (c *client) batch(t *testing.T, id int) {
	suffix := fmt.Sprintf("-%d", id)
	for _, tc := range testCases {
		name := tc.name + suffix
		t.Run(name, func(t *testing.T) {
			in := strings.Replace(tc.in, "-id", suffix, -1)
			if out, err := c.send(in); err != nil || out != tc.out {
				t.Errorf("\nwant: %q got: %q (in=%q / err=%v)", tc.out, out, in, err)
			}
		})
	}
}

type client struct {
	conn net.Conn
}

func randClient(clients []*client) *client {
	rand.Seed(time.Now().Unix())
	return clients[rand.Intn(len(clients))]
}

func newClient(srv *CMD) (*client, error) {
	host := srv.hostname()
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return nil, fmt.Errorf("Failed to open connection to %s: %v", host, err)
	}
	return &client{conn: conn}, nil
}

const UNKNOWN = "UNKNOWN"

func (c *client) send(msg string) (string, error) {
	c.bumpTimeout()
	if _, err := fmt.Fprintln(c.conn, msg); err != nil {
		return UNKNOWN, fmt.Errorf("Error sending message to server: %v", err)
	}

	c.bumpTimeout()
	r, err := bufio.NewReader(c.conn).ReadString('\n')
	if err != nil {
		return UNKNOWN, fmt.Errorf("Error reading response code from server: %v", err)
	}

	return strings.TrimRight(r, "\n"), nil
}

func (c *client) bumpTimeout() {
	c.conn.SetDeadline(time.Now().Add(time.Second * 10))
}

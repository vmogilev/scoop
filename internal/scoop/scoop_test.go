package scoop

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"strings"
	"testing"
	"time"
)

type testCase struct {
	name string
	in   string
	out  string
}

var testCases = []testCase{
	{"MISSING", "QUERY|zmqpp|", FAIL},
	{"ONE", "INDEX|zmqpp|", OK},
	{"NO-DEPS", "INDEX|evas-generic-loaders|aalib,atk,audiofile", FAIL},
	{"TWO", "INDEX|aalib|", OK},
	{"THREE", "INDEX|atk|", OK},
	{"FOUR", "INDEX|audiofile|", OK},
	{"FIVE", "INDEX|audiofile2|", OK},
	{"DEPS", "INDEX|evas-generic-loaders|aalib,atk,audiofile", OK},
	{"SIX", "INDEX|evas-generic-loaders2|aalib,atk,audiofile", OK},
	{"RM-ONE", "REMOVE|zmqpp|", OK},
	{"RM-ONE-2X", "REMOVE|zmqpp|", OK},
	{"RM-FOUR-DEP", "REMOVE|audiofile|", FAIL},
	{"RM-FIVE", "REMOVE|evas-generic-loaders|", OK},
	{"RM-SIX", "REMOVE|evas-generic-loaders2|", OK},
	{"RM-FOUR", "REMOVE|audiofile|", OK},
	{"QUERY-FOUR", "QUERY|audiofile|", FAIL},
	{"QUERY-TWO", "QUERY|aalib|", OK},
}

func TestServer(t *testing.T) {
	port := 9007
	timeoutSecs := 5 * time.Second
	tmp, err := ioutil.TempDir("", "scoop")
	if err != nil {
		t.Fatal(err)
	}
	srv := New(port, timeoutSecs, tmp, true)
	go srv.Start()
	time.Sleep(time.Millisecond * 100)

	defer func() {
		close(srv.kill)
		<-srv.closed
	}()

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

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := randClient(clients)
			if out, err := c.send(tc.in); err != nil || out != tc.out {
				t.Errorf("\nwant: %q got: %q (err=%v)", tc.out, out, err)
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

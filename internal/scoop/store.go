package scoop

import (
	"fmt"
	"log"
	"strings"
)

type store struct {
	dir      string
	messages chan (message)
	log      *log.Logger
}

const (
	sep = "|"
)

// handle passes the input commands to the store worker
func (s *store) handle(input string) (string, error) {
	message, err := parse(strings.Split(input, sep))
	if err != nil {
		return ERROR, err
	}

	s.messages <- message
	r := <-message.retChan
	return r.code, r.err
}

// worker - background worker processing incoming
// messages serially, during shutdown it drains
// the channel and releases waiting go routines
// which are thencollected by listenAndServe()
// through the use of sync.WaitGroup ...
//
// worker is left behind on shutdown (by design)
func (s *store) worker() {
	for m := range s.messages {
		switch m.CMD {
		case INDEX:
			s.index(m)
		case REMOVE:
			s.remove(m)
		case QUERY:
			s.query(m)
		case NOOP:
			m.retChan <- storeResponse{OK, nil}
		default:
			err := fmt.Errorf("message command %q is invalid", m.CMD)
			m.retChan <- storeResponse{ERROR, err}
		}
	}
}

func (s *store) index(msg message) {
	msg.retChan <- storeResponse{OK, nil}
}

func (s *store) remove(msg message) {
	msg.retChan <- storeResponse{OK, nil}
}

func (s *store) query(msg message) {
	msg.retChan <- storeResponse{OK, nil}
}

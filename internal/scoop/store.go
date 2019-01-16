package scoop

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type store struct {
	// PKGS - installed pkgs and their dependencies
	PKGS map[string][]string `json:"pkgs"`

	// DEPS - reverse lookup of dependencies for a given pkg
	// pkg1:{pkg2:true,pkg3:true,pkg4:true}
	// which reads:
	// 	pkgs 2,3,4 depend on pkg 1
	DEPS map[string]map[string]bool `json:"deps"`

	// CreatedAt - The time when the mapping was created (UTC)
	CreatedAt time.Time `json:"createdAt"`

	// UpdatedAt - The time when the mapping was updated last (UTC)
	UpdatedAt time.Time `json:"updatedAt"`

	dir      string
	messages chan (message)
	log      *log.Logger
}

const (
	sep       = "|"
	storeName = "scoop.json"
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
			m.retChan <- response{OK, nil}
		default:
			err := fmt.Errorf("message command %q is invalid", m.CMD)
			m.retChan <- response{ERROR, err}
		}
	}
}

func (s *store) index(msg message) {
	for _, name := range msg.DEP {
		if _, found := s.PKGS[name]; !found {
			err := fmt.Errorf("%q's dependency %q is missing", msg.PKG, name)
			msg.retChan <- response{FAIL, err}
			return
		}
	}

	s.PKGS[msg.PKG] = msg.DEP

	for _, name := range msg.DEP {
		if deps, found := s.DEPS[name]; found {
			deps[msg.PKG] = true
		} else {
			s.DEPS[name] = map[string]bool{msg.PKG: true}
		}
	}

	msg.retChan <- response{OK, nil}
}

func (s *store) remove(msg message) {
	dependsOn, found := s.PKGS[msg.PKG]
	if !found {
		msg.retChan <- response{OK, nil}
		return
	}

	if deps, found := s.DEPS[msg.PKG]; found {
		err := fmt.Errorf("%q has active dependencies %v", msg.PKG, deps)
		msg.retChan <- response{FAIL, err}
		return
	}

	for _, name := range dependsOn {
		deps, found := s.DEPS[name]
		if found {
			delete(deps, msg.PKG)
			if len(deps) == 0 {
				delete(s.DEPS, name)
			}
		}
	}

	delete(s.PKGS, msg.PKG)

	msg.retChan <- response{OK, nil}
}

func (s *store) query(msg message) {
	if _, found := s.PKGS[msg.PKG]; found {
		msg.retChan <- response{OK, nil}
	} else {
		msg.retChan <- response{FAIL, nil}
	}
}

func (s *store) datafile() string {
	path := fmt.Sprintf("%s/%s", s.dir, storeName)
	return filepath.FromSlash(path)
}

// load gets called on startup so it's Ok to explode on failure
// it loads the cached contents of store's datafile into memory
func (s *store) load() {
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		log.Fatalf("ERROR: can't create %q: %v", s.dir, err)
	}

	datafile := s.datafile()

	if _, err := os.Stat(datafile); os.IsNotExist(err) {
		// store does not exist, exiting
		return
	}

	content, err := ioutil.ReadFile(datafile)
	if err != nil {
		log.Fatal(err)
	}

	if err := json.Unmarshal(content, s); err != nil {
		log.Fatalf("ERROR: can't unmarshal datafile %q: %v", datafile, err)
	}

	s.log.Printf("loaded %q datafile", datafile)
}

// unload saves the in-memory store to it's datafile
func (s *store) unload() error {
	s.UpdatedAt = time.Now().UTC()
	b, err := json.MarshalIndent(s, "", "    ")
	if err != nil {
		return err
	}

	datafile := s.datafile()
	s.log.Printf("unloading %q datafile", datafile)
	return ioutil.WriteFile(datafile, b, 0700)
}

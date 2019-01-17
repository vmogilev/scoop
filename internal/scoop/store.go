package scoop

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
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
	storeName = "scoop.json"

	// errors
	errorCodeInvalidCMD = "ERR-0001"
	errorCodeDepMissing = "ERR-0002"
	errorCodeActiveDeps = "ERR-0003"
	errorCodeInvalidPKG = "ERR-0004"
)

// handle passes the input commands to the store worker
func (s *store) handle(input string) (string, error) {
	message, err := parse(input)
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
	var resp response
	for m := range s.messages {
		switch m.CMD {
		case INDEX:
			resp = s.index(m)
		case REMOVE:
			resp = s.remove(m)
		case QUERY:
			resp = s.query(m)
		case NOOP:
			resp = response{OK, nil}
		default:
			err := fmt.Errorf("%s: message command %q is invalid", errorCodeInvalidCMD, m.CMD)
			resp = response{ERROR, err}
		}
		m.retChan <- resp
	}
}

func (s *store) index(msg message) response {
	for _, name := range msg.DEP {
		if _, found := s.PKGS[name]; !found {
			err := fmt.Errorf("%s: %q's dependency %q is missing", errorCodeDepMissing, msg.PKG, name)
			return response{FAIL, err}
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

	return response{OK, nil}
}

func (s *store) remove(msg message) response {
	dependsOn, found := s.PKGS[msg.PKG]
	if !found {
		return response{OK, nil}
	}

	if deps, found := s.DEPS[msg.PKG]; found {
		err := fmt.Errorf("%s: %q has active dependencies %v", errorCodeActiveDeps, msg.PKG, deps)
		return response{FAIL, err}
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

	return response{OK, nil}
}

func (s *store) query(msg message) response {
	if _, found := s.PKGS[msg.PKG]; found {
		return response{OK, nil}
	}
	err := fmt.Errorf("%s: %q isn't indexed", errorCodeInvalidPKG, msg.PKG)
	return response{FAIL, err}
}

func (s *store) datafile() string {
	path := fmt.Sprintf("%s/%s", s.dir, storeName)
	return filepath.FromSlash(path)
}

func (s *store) lockfile() string {
	return s.datafile() + ".lock"
}

// load gets cached contents of store's datafile into memory
// and creates a lock to ensure no other instance of it can do so
func (s *store) load() error {
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return fmt.Errorf("can't create %q: %v", s.dir, err)
	}

	datafile := s.datafile()
	lockfile := s.lockfile()

	if _, err := os.Stat(lockfile); err == nil {
		return fmt.Errorf("can't load store from %q - it's locked", s.dir)
	}

	s.log.Printf("creating lock file: %q", lockfile)
	pid := fmt.Sprintf("%d", os.Getpid())
	if err := ioutil.WriteFile(lockfile, []byte(pid), 0700); err != nil {
		return fmt.Errorf("can't create lockfile %q: %v", lockfile, err)
	}

	if _, err := os.Stat(datafile); os.IsNotExist(err) {
		// store does not exist, nothing to load, and
		// no reason to create, it'll get created on exit
		return nil
	}

	content, err := ioutil.ReadFile(datafile)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(content, s); err != nil {
		return fmt.Errorf("can't unmarshal datafile %q: %v", datafile, err)
	}

	s.log.Printf("loaded %q datafile", datafile)
	return nil
}

// unload saves the in-memory store to it's datafile
func (s *store) unload() error {
	s.UpdatedAt = time.Now().UTC()
	b, err := json.MarshalIndent(s, "", "    ")
	if err != nil {
		return err
	}

	datafile := s.datafile()
	lockfile := s.lockfile()
	s.log.Printf("unloading %q datafile", datafile)
	if err := ioutil.WriteFile(datafile, b, 0700); err != nil {
		return err
	}

	return os.Remove(lockfile)
}

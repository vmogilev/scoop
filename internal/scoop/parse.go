package scoop

import (
	"fmt"
	"strings"
)

type message struct {
	CMD     string
	PKG     string
	DEP     []string
	retChan chan (response)
}

type response struct {
	code string
	err  error
}

func parse(m []string) (message, error) {
	if len(m) != 3 {
		return message{}, fmt.Errorf("message format is invalid: expecting 3 got %d tokens", len(m))
	}

	cmd := m[0]
	pkg := m[1]
	dep := m[2]

	return message{cmd, pkg, strings.Split(dep, ","), make(chan response)}, nil
}

package scoop

import (
	"reflect"
	"testing"
)

func TestParser(t *testing.T) {
	testCases := []struct {
		name string
		in   string
		out  message
		err  bool
	}{
		{"INDEX/DEP", "INDEX|cloog|gmp,isl,pkg-config", message{"INDEX", "cloog", []string{"gmp", "isl", "pkg-config"}, nil}, false},
		{"INDEX/NO-DEP", "INDEX|ceylon|", message{"INDEX", "ceylon", []string{}, nil}, false},
		{"REMOVE", "REMOVE|cloog|", message{"REMOVE", "cloog", []string{}, nil}, false},
		{"QUERY", "QUERY|cloog|", message{"QUERY", "cloog", []string{}, nil}, false},
		{"MISSING-PIPE", "QUERY|cloog", message{}, true},
		{"EXTRA-PIPE", "INDEX|cloog|gmp,isl,pkg-config|", message{}, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := parse(tc.in)
			if tc.err && err == nil {
				t.Fatalf("\nwant error got none")
			}
			out.retChan = tc.out.retChan
			if !reflect.DeepEqual(tc.out, out) {
				t.Fatalf("\nwant: %#v\ngot: %#v", tc.out, out)
			}
		})
	}
}

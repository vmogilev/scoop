package scoop

import (
	"errors"
	"log"
	"os"
	"strings"
	"testing"
)

func TestStoreCMDs(t *testing.T) {
	ne := func(s string) error { return errors.New(s) }

	testCases := []struct {
		name string
		in   string
		out  response
	}{
		{"BAD", "JUNK|berkeley-db4|", response{ERROR, ne(errorCodeInvalidCMD)}},
		{"NOOP", "NOOP||", response{OK, nil}},
		{"ONE", "INDEX|berkeley-db4|", response{OK, nil}},
		{"DUP", "INDEX|berkeley-db4|", response{OK, nil}},
		{"MISSING-DEPS", "INDEX|evas-generic-loaders|aalib,atk,audiofile", response{FAIL, ne(errorCodeDepMissing)}},
		{"TWO", "INDEX|aalib|", response{OK, nil}},
		{"THREE", "INDEX|atk|", response{OK, nil}},
		{"FOUR", "INDEX|audiofile|", response{OK, nil}},
		{"FIVE", "INDEX|evas-generic-loaders|aalib,atk,audiofile", response{OK, nil}},
		{"SIX", "INDEX|evas-generic-loaders2|aalib,atk,audiofile", response{OK, nil}},
		{"RM-ONE", "REMOVE|berkeley-db4|", response{OK, nil}},
		{"RM-ONE-2X", "REMOVE|berkeley-db4|", response{OK, nil}},
		{"RM-FOUR-DEP", "REMOVE|audiofile|", response{FAIL, ne(errorCodeActiveDeps)}},
		{"RM-FIVE", "REMOVE|evas-generic-loaders|", response{OK, nil}},
		{"RM-SIX", "REMOVE|evas-generic-loaders2|", response{OK, nil}},
		{"RM-FOUR", "REMOVE|audiofile|", response{OK, nil}},
		{"QUERY-FOUR", "QUERY|audiofile|", response{FAIL, ne(errorCodeInvalidPKG)}},
		{"QUERY-TWO", "QUERY|aalib|", response{OK, nil}},
	}

	log := log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.LUTC|log.Lshortfile)
	s := &store{
		PKGS:     make(map[string][]string),
		DEPS:     make(map[string]map[string]bool),
		messages: make(chan message),
		log:      log,
	}

	go s.worker()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var resp response
			resp.code, resp.err = s.handle(tc.in)

			if tc.out.code != resp.code {
				t.Errorf("invalid response code, want: %q, got: %q", tc.out.code, resp.code)
			}

			if tc.out.err != nil && (resp.err == nil || !strings.HasPrefix(resp.err.Error(), tc.out.err.Error())) {
				t.Errorf("invalid response error, want: %v, got: nil", tc.out.err)
			}

			if tc.out.err == nil && resp.err != nil {
				t.Errorf("invalid response error, want: nil, got: %v", resp.err)
			}
		})
	}
}

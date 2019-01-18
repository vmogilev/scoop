package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/vmogilev/scoop/internal/scoop"
)

var (
	semver string
	relver string
)

func main() {
	port := flag.Int("port", 8080, "The port to listen on")
	timeout := flag.Int64("timeout", 120, "Idle Connection Timeout in Seconds")
	dir := flag.String("dir", "./scoop-data", "Location of scoop's data directory")
	verbose := flag.Bool("verbose", false, "Set this to `true` for verbose output")
	version := flag.Bool("version", false, "Set this to `true` to get current version")
	flag.Parse()

	if *version {
		fmt.Printf("scoop: %s-%s\n", semVer(), relVer())
		return
	}

	timeoutSecs := time.Duration(*timeout) * time.Second
	scoop.New(*port, timeoutSecs, *dir, *verbose).Start()
}

func semVer() string {
	if semver != "" {
		return semver
	}
	return "devel"
}

func relVer() string {
	if relver != "" {
		return relver
	}
	return "devel"
}

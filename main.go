package main

import (
	"flag"
	"time"

	"github.com/vmogilev/scoop/internal/scoop"
)

func main() {
	port := flag.Int("port", 8080, "The port to listen on")
	timeout := flag.Int64("timeout", 60, "Idle Connection Timeout in Seconds")
	dir := flag.String("dir", "./scoop-data", "Location of scoop's data directory")
	verbose := flag.Bool("verbose", false, "Set this to `true` for verbose output")
	flag.Parse()

	timeoutSecs := time.Duration(*timeout) * time.Second
	scoop.New(*port, timeoutSecs, *dir, *verbose).Start()
}

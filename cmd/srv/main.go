package main

import (
	"flag"
	"fmt"
	"os"

	"srv.exe.dev/srv"
)

var flagListenAddr = flag.String("listen", ":8000", "address to listen on")

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	flag.Parse()
	if env := os.Getenv("LISTEN_ADDR"); env != "" && !isFlagSet("listen") {
		*flagListenAddr = env
	}
	if env := os.Getenv("PORT"); env != "" && !isFlagSet("listen") && os.Getenv("LISTEN_ADDR") == "" {
		*flagListenAddr = ":" + env
	}
	server, err := srv.New("db.sqlite3")
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}
	return server.Serve(*flagListenAddr)
}

func isFlagSet(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

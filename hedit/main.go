package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println("HEdit v0.1 — github.com/iwohost/hedit")
		return
	}

	e, err := NewEditor(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "hedit: %v\n", err)
		os.Exit(1)
	}

	if err := e.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "hedit: %v\n", err)
		os.Exit(1)
	}
}

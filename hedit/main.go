package main

import (
	"fmt"
	"os"
)

func main() {
	filename := ""
	if len(os.Args) > 1 {
		if os.Args[1] == "--version" || os.Args[1] == "-v" {
			fmt.Println("HEdit v0.1 — github.com/iwohost/hedit")
			return
		}
		filename = os.Args[1]
	}

	e, err := NewEditor(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hedit: %v\n", err)
		os.Exit(1)
	}

	if err := e.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "hedit: %v\n", err)
		os.Exit(1)
	}
}

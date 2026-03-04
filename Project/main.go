package main

import (
	"fmt"
	"os"

	tester "github.com/KralHa0/TTK4145_16/Project/Tester"
)

func main() {
	fmt.Println("Starting...")
	if len(os.Args) < 2 {
		printUsage()
		return
	}
	switch os.Args[1] {
	case "nw":
		tester.RunNwTest()
	case "om":
		tester.RunOMTest()
	case "run":
		tester.RunTest()
	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Println("Usage: ./main <test>")
	fmt.Println("  nw   — network test (single machine)")
	fmt.Println("  om   — order manager test (single machine, hardcoded)")
	fmt.Println("  run  — combined network + updater test (multi machine)")
}

package main

import (
	"fmt"
	"os"

	"github.com/liranbh7/huh/internal/classify"
	"github.com/liranbh7/huh/internal/pid"
	"github.com/liranbh7/huh/internal/port"
	"github.com/liranbh7/huh/internal/print"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: huh <port | pid | process | path | binary>\n")
		os.Exit(1)
	}
	input := os.Args[1]
	types := classify.Classify(input)

	printed := 0
	for _, t := range types {
		switch t {
		case classify.Port:
			r, err := port.Resolve(input)
			if err != nil {
				if len(types) == 1 { // only print the error if this is the only type we found
					printError(err)
				}
				continue
			}
			sep(printed)
			print.Port(r)
			printed++
		case classify.PID:
			r, err := pid.Resolve(input)
			if err != nil {
				if len(types) == 1 { // only print the error if this is the only type we found
					printError(err)
				}
				continue
			}
			sep(printed)
			print.PID(r)
			printed++
		case classify.ProcessName:
			printError(fmt.Errorf("process resolver not yet implemented"))
		case classify.Path:
			printError(fmt.Errorf("device/path resolver not yet implemented"))
		case classify.Binary:
			printError(fmt.Errorf("binary resolver not yet implemented"))
		case classify.Unknown:
			printError(fmt.Errorf("huh: don't know what %q is", input))
		}
	}
}

// sep prints a blank line between multiple results.
func sep(printed int) {
	if printed > 0 {
		fmt.Println()
	}
}

func printError(err error) {
	fmt.Fprintf(os.Stderr, "huh: %s\n", err)
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "huh: %s\n", err)
	os.Exit(1)
}

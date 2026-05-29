package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/liranbh7/huh/src/binary"
	"github.com/liranbh7/huh/src/classify"
	"github.com/liranbh7/huh/src/device"
	"github.com/liranbh7/huh/src/pid"
	"github.com/liranbh7/huh/src/port"
	"github.com/liranbh7/huh/src/print"
	"github.com/liranbh7/huh/src/processname"
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
				continue
			}
			sep(printed)
			print.Port(r)
			printed++
		case classify.PID:
			r, err := pid.Resolve(input)
			if err != nil {
				continue
			}
			sep(printed)
			print.PID(r)
			printed++
		case classify.ProcessName:
			r, err := processname.Resolve(input)
			if err != nil {
				continue
			}
			sep(printed)
			print.ProcessName(r)
			printed++
		case classify.Path:
			r, err := device.Resolve(input)
			if err != nil {
				continue
			}
			sep(printed)
			print.Device(r)
			printed++
		case classify.Binary:
			r, err := binary.Resolve(input)
			if err != nil {
				continue
			}
			sep(printed)
			print.Binary(r)
			printed++
		case classify.Unknown:
			printError(fmt.Errorf("huh: don't know what %q is", input))
		}
	}

	// if we found some types for this input but none of them yielded results, print an error with the types we tried
	// if no types were found, the error will already have been printed by classify.Unknown, so we don't need to print another error in that case
	if printed == 0 && len(types) > 0 && types[0] != classify.Unknown {
		// types print is in format of [port, pid]
		var typesStr []string
		for _, t := range types {
			typesStr = append(typesStr, t.String())
		}

		printError(fmt.Errorf("huh: no results found for %q in [%v]", input, strings.Join(typesStr, ", ")))

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

package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("USAGE: %v VERSION\n", os.Args[0])
		os.Exit(1)
	}

	version := os.Args[1]
	if err := run(version, os.Stdout); err != nil {
		log.Fatal(err)
	}
}

const (
	foundVersionNumber = iota + 1
	foundHeader

	printing
	foundEmptyLine
)

func run(version string, out io.Writer) error {
	file, err := os.OpenFile("CHANGELOG.md", os.O_RDONLY, 0444)
	if err != nil {
		return err
	}
	defer file.Close()

	state := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		switch state {
		case 0:
			if strings.HasPrefix(line, version+" ") {
				state = foundVersionNumber
			}
		case foundVersionNumber:
			// ---- below the version number
			if strings.Trim(line, "-") == "" {
				state = foundHeader
			} else {
				return fmt.Errorf("unexpected %q after version number", line)
			}
		case foundHeader:
			if line == "" {
				continue
			}
			state = printing
			fallthrough
		case printing:
			if strings.HasPrefix(line, "v") {
				// end of section
				return nil
			}
			if line == "" {
				state = foundEmptyLine
				continue
			}
			fmt.Fprintln(out, line)
		case foundEmptyLine:
			if line == "" {
				// two empty lines end the changelog
				return nil
			}
			fmt.Println()
			fmt.Fprintln(out, line)
			state = printing
		default:
			return fmt.Errorf("unexpected state %v on line %q", state, line)
		}

	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if state < printing {
		return fmt.Errorf("could not find version %q in changelog", version)
	}
	return nil
}

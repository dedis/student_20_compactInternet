package shell

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Shell represent the command console
type Shell struct {
	LineSymbol        string
	ArgumentSeparator string
	reader            *bufio.Reader
}

// Red text format
const Red string = "\033[31m"

// Yellow text format
const Yellow string = "\033[33m"

// Blue text format
const Blue string = "\033[34m"

// Green text format
const Green string = "\033[32m"

// Clear previous formats
const Clear string = "\033[0m"

// GetCommand displays a shell line and parses the input
func (s *Shell) GetCommand() []string {
	fmt.Print(s.LineSymbol)
	fmt.Print(" ")
	cmdString, err := s.reader.ReadString('\n')
	cmdString = strings.ReplaceAll(cmdString, "\r", "")
	cmdString = strings.ReplaceAll(cmdString, "\n", "")
	if err != nil {
		panic("Could not get command")
	}
	return strings.Split(cmdString, s.ArgumentSeparator)
}

func (s *Shell) Write(payload string) {
	fmt.Print(payload)
}

var maxOverwritten int = 0

// Overwrite allows to write over the current console line
func (s *Shell) Overwrite(payload ...string) {
	fmt.Print("\r \r")

	var pLength int
	for p := range payload {
		s.Write(payload[p])
		pLength += len(payload[p])
	}

	if pLength > maxOverwritten {
		maxOverwritten = pLength
	} else {
		s.Write(strings.Repeat("   ", maxOverwritten-pLength))
	}
}

// InitShell creates a Shell
func InitShell(symbol string, separator string) *Shell {
	shell := Shell{LineSymbol: symbol, ArgumentSeparator: separator, reader: bufio.NewReader(os.Stdin)}
	return &shell
}

package main

import (
	"bufio"
	"fmt"
	"os"
	"unicode/utf8"

	"golang.org/x/term"
)

type keyEvent struct {
	Code    rune
	Display string
}

type consoleReader struct {
	oldState *term.State
	reader   *bufio.Reader
}

func newConsoleReader() (*consoleReader, error) {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return nil, err
	}

	return &consoleReader{
		oldState: oldState,
		reader:   bufio.NewReader(os.Stdin),
	}, nil
}

func (r *consoleReader) Close() error {
	if r.oldState == nil {
		return nil
	}
	return term.Restore(int(os.Stdin.Fd()), r.oldState)
}

func (r *consoleReader) ReadKey() (keyEvent, error) {
	char, _, err := r.reader.ReadRune()
	if err != nil {
		return keyEvent{}, err
	}

	if char == utf8.RuneError {
		return keyEvent{Code: char, Display: "<INVALID-UTF8>"}, nil
	}
	if char < 32 || char == 127 {
		return keyEvent{Code: char, Display: fmt.Sprintf("<CTRL-%d>", char)}, nil
	}
	return keyEvent{Code: char, Display: string(char)}, nil
}

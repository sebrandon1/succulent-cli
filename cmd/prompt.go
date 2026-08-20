package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

var (
	promptReader     io.Reader = os.Stdin
	promptWriter     io.Writer = os.Stderr
	promptLineReader *bufio.Reader
	isInteractive    = stdinIsTerminal
)

func stdinIsTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	return fi.Mode()&os.ModeCharDevice != 0
}

func promptLine(prompt string) (string, error) {
	if _, err := fmt.Fprint(promptWriter, prompt); err != nil {
		return "", fmt.Errorf("writing prompt: %w", err)
	}

	if promptLineReader == nil {
		promptLineReader = bufio.NewReader(promptReader)
	}

	line, err := promptLineReader.ReadString('\n')
	if err == nil || (err == io.EOF && line != "") {
		return strings.TrimSpace(line), nil
	}

	if err == io.EOF {
		return "", fmt.Errorf("reading prompt: EOF")
	}

	return "", fmt.Errorf("reading prompt: %w", err)
}

func promptIfEmpty(value, requiredErr, prompt string) (string, error) {
	if value != "" {
		return value, nil
	}

	if !isInteractive() {
		return "", errors.New(requiredErr)
	}

	line, err := promptLine(prompt)
	if err != nil {
		return "", err
	}

	if line == "" {
		return "", errors.New(requiredErr)
	}

	return line, nil
}

func promptOptional(value string) (string, error) {
	if value != "" || !isInteractive() {
		return value, nil
	}

	return promptLine(promptOCPTag)
}

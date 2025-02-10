package main

import (
	"bytes"
	"errors"
	"net"
	"os/exec"
	"strings"
)

func runCommand(name string, args ...string) (int, string) {
	cmd := exec.Command(name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			return -1, err.Error()
		}
	}

	outputLines := strings.Split(out.String(), "\n")
	if len(outputLines) > 5 {
		outputLines = outputLines[len(outputLines)-5:]
	}

	return exitCode, strings.Join(outputLines, "\n")
}

func availablePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer func() { _ = l.Close() }()

	return l.Addr().(*net.TCPAddr).Port, nil
}

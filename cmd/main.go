package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/xconnio/xconn-go"
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

func main() {
	if len(os.Args) < 2 {
		log.Fatalln("Must be called with at least one argument")
	}

	command := os.Args[1]
	args := os.Args[2:]
	exitCode, output := runCommand(command, args...)

	fmt.Printf("Exit Code: %d\nOutput:\n%s", exitCode, output)

	session, err := xconn.Connect(context.Background(), "ws://localhost:8080/ws", "realm1")
	if err != nil {
		log.Fatal(err)
	}

	if err := session.Publish("io.xconn.output", []any{exitCode, output}, nil, nil); err != nil {
		log.Fatal(err)
	}
}

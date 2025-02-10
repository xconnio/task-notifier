package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"

	"github.com/xconnio/xconn-go"
)

const TaskNotifierRealm = "io.xconn.task_notifier"

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

func main() {
	if len(os.Args) < 2 {
		log.Fatalln("Must be called with at least one argument")
	}

	rout := xconn.NewRouter()
	rout.AddRealm(TaskNotifierRealm)

	port, err := availablePort()
	if err != nil {
		log.Fatalln(err)
	}

	server := xconn.NewServer(rout, nil, nil)
	closer, err := server.Start("0.0.0.0", port)
	if err != nil {
		log.Fatalln(err)
	}
	defer closer.Close()

	command := os.Args[1]
	args := os.Args[2:]
	exitCode, output := runCommand(command, args...)

	fmt.Printf("Exit Code: %d\nOutput:\n%s", exitCode, output)

	session, err := xconn.Connect(context.Background(), fmt.Sprintf("ws://localhost:%v/ws", port), TaskNotifierRealm)
	if err != nil {
		log.Fatal(err)
	}

	if err := session.Publish("io.xconn.output", []any{exitCode, output}, nil,
		map[string]any{"acknowledge": true}); err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/xconnio/xconn-go"
)

const TaskNotifierRealm = "io.xconn.task_notifier"

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

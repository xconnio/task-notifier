package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/grandcat/zeroconf"

	"github.com/xconnio/wampproto-go/util"
	"github.com/xconnio/xconn-go"
)

const TaskNotifierRealm = "io.xconn.task_notifier"

func main() {
	flag.Parse()

	waitTime, cliArgs := extractWaitFlag(flag.Args())
	if len(cliArgs) < 1 {
		log.Fatalln("Must be called with at least one argument")
	}

	if waitTime != 0 {
		resolver, err := zeroconf.NewResolver(nil)
		if err != nil {
			log.Fatalln("Failed to initialize resolver:", err.Error())
		}

		entries := make(chan *zeroconf.ServiceEntry)
		foundProcess := make(chan int)
		go func(results <-chan *zeroconf.ServiceEntry) {
			for entry := range results {
				log.Println(entry.HostName, entry.Text, entry.Port)
				if waitTime == entry.Port {
					foundProcess <- entry.Port
					return
				}
			}
		}(entries)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
		defer cancel()
		err = resolver.Browse(ctx, "_tasknotifier._tcp", "local.", entries)
		if err != nil {
			log.Fatalln("Failed to browse:", err)
		}

		doneChan := make(chan struct{})
		select {
		case port := <-foundProcess:
			session, err := xconn.Connect(context.Background(), fmt.Sprintf("ws://localhost:%v/ws", port),
				TaskNotifierRealm)
			if err != nil {
				log.Fatalln(err)
			}

			_, err = session.Subscribe("io.xconn.output", func(event *xconn.Event) {
				exitCode, ok := util.AsInt64(event.Arguments[0])
				if !ok {
					log.Fatalln("Failed to parse exit code:", event.Arguments[0])
					return
				}
				if exitCode != 0 {
					log.Fatalf("Command was not successful: exit code: %v, logs: %s", exitCode,
						event.Arguments[1])
				}

				if err := runCommandAndPublish(cliArgs); err != nil {
					log.Fatalln(err)
				}
				doneChan <- struct{}{}
			}, nil)
			if err != nil {
				log.Fatalln(err)
			}

		case <-ctx.Done():
			log.Fatalln("Timeout: Could not find expected process")
		}

		<-doneChan
	} else {
		log.Fatalln(runCommandAndPublish(cliArgs))
	}
}

func runCommandAndPublish(cliArgs []string) error {
	rout := xconn.NewRouter()
	rout.AddRealm(TaskNotifierRealm)

	port, err := availablePort()
	if err != nil {
		return err
	}

	server := xconn.NewServer(rout, nil, nil)
	closer, err := server.Start("0.0.0.0", port)
	if err != nil {
		return err
	}
	defer closer.Close()

	command := cliArgs[0]
	args := cliArgs[1:]

	hostname, _ := os.Hostname()
	mdns, err := zeroconf.Register(hostname, "_tasknotifier._tcp", "local.", port,
		[]string{fmt.Sprintf("%v", port)}, nil)
	if err != nil {
		return err
	}
	defer mdns.Shutdown()

	exitCode, output := runCommand(command, args...)

	fmt.Printf("Exit Code: %d\nOutput:\n%s", exitCode, output)

	session, err := xconn.Connect(context.Background(), fmt.Sprintf("ws://localhost:%v/ws", port), TaskNotifierRealm)
	if err != nil {
		return err
	}

	return session.Publish("io.xconn.output", []any{exitCode, output}, nil,
		map[string]any{"acknowledge": true})
}

// extractWaitFlag extracts the value of --wait and removes it from the arguments.
func extractWaitFlag(args []string) (int, []string) {
	var filteredArgs []string
	waitTime := 0

	for i := 0; i < len(args); i++ {
		if args[i] == "--wait" && i+1 < len(args) {
			val, err := strconv.Atoi(args[i+1])
			if err == nil {
				waitTime = val
			}
			i++
		} else {
			filteredArgs = append(filteredArgs, args[i])
		}
	}
	return waitTime, filteredArgs
}

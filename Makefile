build:
	go build github.com/xconnio/task-notifier/cmd/tasknotifer

lint:
	golangci-lint run

test:
	go test -count=1 ./... -v

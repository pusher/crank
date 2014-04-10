
all: fmt crank

fmt:
	go fmt ./...

crank: **/*.go
	cd cmd/crank && go build -o ../../$@

.PHONY: all fmt

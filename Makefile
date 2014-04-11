
all: fmt crank

fmt:
	go fmt ./...

clean:
	rm -f crank

crank: **/*.go
	cd cmd/crank && go build -o ../../$@

.PHONY: all fmt clean

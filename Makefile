
all: fmt crank

fmt:
	go fmt ./...

clean:
	rm -f crank

crank: cmd/crank/*.go pkg/**/*.go
	cd cmd/crank && go build -o ../../$@

.PHONY: all fmt clean

GOFLAGS=

all: fmt crank crankctl

fmt:
	go fmt ./...

clean:
	rm -f crank crankctl

crank: cmd/crank/*.go pkg/**/*.go
	cd cmd/$@ && go build $(GOFLAGS) -o ../../$@

crankctl: cmd/crankctl/*.go pkg/**/*.go
	cd cmd/$@ && go build $(GOFLAGS) -o ../../$@

.PHONY: all fmt clean

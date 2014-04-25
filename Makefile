GOFLAGS=
GOREV=-ldflags "-X main.build \"SHA: $(shell git rev-parse HEAD) (Built $(shell date) with $(shell go version))\""

all: fmt crank crankctl

fmt:
	go fmt ./...

clean:
	rm -f crank crankctl

crank: cmd/crank/*.go pkg/**/*.go
	go build $(GOREV) $(GOFLAGS) -o $@ ./cmd/$@

crankctl: cmd/crankctl/*.go pkg/**/*.go
	go build $(GOREV) $(GOFLAGS) -o $@ ./cmd/$@

.PHONY: all fmt clean

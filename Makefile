GOFLAGS=
GOREV:=-ldflags "-X main.build \"SHA: $(shell git rev-parse HEAD) (Built $(shell date) with $(shell go version))\""
PREFIX=/usr/local

all: fmt test crank crankctl

test:
	go test ./...

install: crank crankctl
	install -d $(PREFIX)/bin
	install -d $(PREFIX)/share/man/man1
	install crank $(PREFIX)/bin/crank
	install crankctl $(PREFIX)/bin/crankctl
	install crankx $(PREFIX)/bin/crankx
	cp -R man/*.1 $(PREFIX)/share/man/man1

fmt:
	go fmt ./...

clean:
	rm -f crank crankctl

crank: cmd/crank/*.go src/**/*.go
	go build $(GOREV) $(GOFLAGS) -o $@ ./cmd/$@

crankctl: cmd/crankctl/*.go src/**/*.go
	go build $(GOREV) $(GOFLAGS) -o $@ ./cmd/$@

.PHONY: all fmt clean install test

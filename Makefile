
all: fmt crank crankctl

fmt:
	go fmt ./...

clean:
	rm -f crank

crank: cmd/crank/*.go pkg/**/*.go
	cd cmd/$@ && go build -o ../../$@

crankctl: cmd/crankctl/*.go pkg/**/*.go
	cd cmd/$@ && go build -o ../../$@

.PHONY: all fmt clean

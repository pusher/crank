
crank: *.go
	go fmt ./...
	go build -o $@

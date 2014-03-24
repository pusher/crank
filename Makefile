
crank: *.go logfile/*.go
	go fmt ./...
	go build -o $@

package main

import (
	"flag"
	"log"
	"net/rpc"
)

var run string

func init() {
	flag.StringVar(&run, "run", "", "path to control socket")
}

func main() {
	flag.Parse()

	client, err := rpc.Dial("unix", run)
	if err != nil {
		log.Fatal("Couldn't connect: ", err)
	}

	msg := flag.Arg(0)
	var reply string

	err = client.Call("crank.Echo", msg, &reply)
	if err != nil {
		log.Fatal("echo error:", err)
	}
	log.Println("echo reply: ", reply)
}

package main

import (
	"flag"
	"log"
)

func main() {
	var addr = flag.String("addr", "", "external address to bind (e.g. ':80')")
	flag.Parse()

	if len(*addr) == 0 {
		log.Fatal("Missing required flag: addr")
	}

	external := NewExternal(*addr)

	log.Print(external)
}

package main

import (
	"os"
)

type Prototype struct {
	cmd  string
	args []string
	fd   *os.File
}

func NewPrototype(cmd string, args []string, fd *os.File) *Prototype {
	return &Prototype{cmd, args, fd}
}

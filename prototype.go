package main

import (
	"io"
	"os"
)

type Prototype struct {
	cmd  string
	args []string
	fd   *os.File
	out  io.Writer
}

func NewPrototype(cmd string, args []string, fd *os.File, out io.Writer) *Prototype {
	return &Prototype{cmd, args, fd, out}
}

package devnull

import (
	"os"
)

var file *os.File
var err error

func init() {
	file, err = os.Open("/dev/null")
}

func File() (*os.File, error) {
	return file, err
}

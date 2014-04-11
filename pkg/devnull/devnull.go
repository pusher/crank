package devnull

import (
	"os"
)

var File *os.File

func init() {
	var err error
	if File, err = os.Open("/dev/null"); err != nil {
		panic("could not open /dev/null: " + err.Error())
	}
}

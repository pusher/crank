package crank

import (
	"fmt"
	"log"
	"runtime"
	"time"
)

// Used as an alternative to time.After() to never get a timeout on a
// channel select.
var neverChan <-chan time.Time

func init() {
	neverChan = make(chan time.Time)
}

// Used in dark corners of the app where behavior is undefined.
//
// We don't really want to shutdown crank but at least we can show some more
// context.
func fail(v ...interface{}) (err error) {
	_, file, line, _ := runtime.Caller(1)

	args := make([]interface{}, 2, len(v)+2)
	args[0] = file
	args[1] = line
	args = append(args, v...)

	err = fmt.Errorf("ERROR at %s:%d. ", args...)

	log.Println(err)

	return err
}

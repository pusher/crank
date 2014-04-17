package crank

import (
	"fmt"
	"log"
	"runtime"
)

// Used in dark corners of the app where behavior is undefined.
//
// We don't really want to shutdown crank but at least we can show some more
// context.
func fail(v ...interface{}) (err error) {
	_, file, line, _ := runtime.Caller(1)

	data := fmt.Sprintf("", v...)

	err = fmt.Errorf("ERROR at %s:%d. %s", file, line, data)

	log.Println(err)

	return err
}

package crank

import (
	"testing"
)

func TestProcessConfigCloning(t *testing.T) {
	c := &ProcessConfig{"hello", []string{"world"}, 1, 2}

	c2 := c.clone()
	c2.Command = "bob"

	if c == c2 {
		t.Fail()
	}

}

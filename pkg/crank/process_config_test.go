package crank

import (
	"testing"
)

func TestProcessConfigCloning(t *testing.T) {
	c := &ProcessConfig{"hello", []string{"world"}, 1, 2}

	c2 := c.clone()
	c2.Command = []string{"bob"}

	if c == c2 {
		t.Error(c, c2)
	}

	if c2.StartTimeout != 1 {
		t.Error("start timeout", c2.StartTimeout)
	}

}

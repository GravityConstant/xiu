package util

import (
	"testing"
)

func TestBlueColor(t *testing.T) {
	Trace("hello %s", "word")
	Info("hello %s", "word")
	Warning("hello %s", "word")
	Error("hello %s", "word")
	Success("hello %s", "word")
}

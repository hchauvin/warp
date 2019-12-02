package log

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"os/exec"
	"testing"
)

func TestLevels(t *testing.T) {
	w := &bytes.Buffer{}
	l := &Logger{
		Writer: w,
	}

	l.Info("foo", "bar %d", 10)
	assert.Equal(t, "foo: bar 10\n", string(w.Bytes()))
	w.Reset()

	l.Warning("foo", "bar %d", 10)
	assert.Equal(t, "foo: WARNING: bar 10\n", string(w.Bytes()))
	w.Reset()

	l.Error("foo", "bar %d", 10)
	assert.Equal(t, "foo: ERROR: bar 10\n", string(w.Bytes()))
	w.Reset()
}

func TestPipe(t *testing.T) {
	l := &Logger{}

	cmd := exec.Command("foo", "bar")
	cmd.Stdout = &bytes.Buffer{}
	assert.Panics(t, func() { l.Pipe("domain", cmd) })

	cmd = exec.Command("foo", "bar")
	cmd.Stderr = &bytes.Buffer{}
	assert.Panics(t, func() { l.Pipe("domain", cmd) })
}

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoggerSugar(t *testing.T) {
	l := logger{"test", "test"}
	o, err := l.Write([]byte("test"))
	assert.NoError(t, err)
	assert.Equal(t, 4, o)
}

func TestLogger(t *testing.T) {
	l := logger{"test", "test"}
	o, err := l.Write([]byte("{test}"))
	assert.NoError(t, err)
	assert.Equal(t, 6, o)
}

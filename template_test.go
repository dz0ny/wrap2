package main

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var tmpl = "ENV var example custom\n"

func TestDefaultTemplate(t *testing.T) {
	tm := Template{
		Source: "fixtures/in.tmpl",
		Target: "/tmp/target.tmpl",
	}
	os.Setenv("CUSTOM", "custom")
	err := tm.Process()
	assert.NoError(t, err)
	data, err := ioutil.ReadFile(tm.Target)
	assert.NoError(t, err)
	assert.Equal(t, string(data), tmpl)
}

func TestHelpers(t *testing.T) {
	h := sha("test")
	assert.Equal(t, h, "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3")
}

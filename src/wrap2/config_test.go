package main

import (
	"fmt"
	"testing"

	toml "github.com/pelletier/go-toml"
	"github.com/stretchr/testify/assert"
)

var out = `
[[process]]
  cmd = "nginx -V -E"
  [process.config]
    src = "source.tmpl"
    dst = "target.tmpl"
		[process.config.data]
			domain = "test.tld"

[[process]]
  cmd = "php -v"
  [process.config]
    src = "source.tmpl"
    dst = "target.tmpl"

[[process]]
  cmd = "true -v"
`

func TestDefaultConfig(t *testing.T) {
	c := Config{
		PreStart: Command{
			Command: "/bin/bash /provision/bin/initialize",
			Template: Template{
				Source: "source.tmpl",
				Target: "target.tmpl",
			},
		},
		Process: []Command{
			{
				Command: "nginx -V -E",
				Template: Template{
					Source: "source.tmpl",
					Target: "target.tmpl",
					Context: map[string]string{
						"domain": "test.tld",
					},
				},
			},
			{
				Command: "php -v",
				Template: Template{
					Source: "source.tmpl",
					Target: "target.tmpl",
				},
			},
			{
				Command: "true -v",
			},
		},
	}

	err := toml.Unmarshal([]byte(out), &Config{})
	assert.NoError(t, err)

	out, err := toml.Marshal(c)
	fmt.Println(string(out))
	assert.NoError(t, err)
	assert.Contains(t, string(out), `src = "source.tmpl"`)
	assert.Contains(t, string(out), `dst = "target.tmpl"`)
	assert.Contains(t, string(out), `cmd = "nginx -V -E"`)
	assert.Contains(t, string(out), `cmd = "php -v"`)
	assert.Contains(t, string(out), `[process.config]`)
	assert.Contains(t, string(out), `[pre_start]`)
}

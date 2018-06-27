package main

import (
	"io/ioutil"
	"os"

	"github.com/pelletier/go-toml"
	"go.uber.org/zap"
)

// Config is top level init configuration holder
type Config struct {
	PreStart  Command   `toml:"pre_start, omitempty"`
	PostStart Command   `toml:"pre_start, omitempty"`
	Process   []Command `toml:"process"`
}

// NewConfig returns Config instance for provided file
func NewConfig(path string) *Config {
	c := &Config{}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Fatal("unable to open", zap.String("config", path), zap.Error(err))
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal("Config cannot be read", zap.Error(err))
	}

	if err = toml.Unmarshal(data, c); err != nil {
		log.Fatal("Config cannot be parsed", zap.Error(err))
	}

	return c
}

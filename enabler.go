package main

import (
	"fmt"
	"os"
	"strings"
)

type Enabler struct {
	Key      string `toml:"key"`
	Operator string `toml:"operator"`
	Value    string `toml:"value"`
}

// IsTrue evaluates the expression
func (t *Enabler) IsTrue() bool {
	if t.Operator == "EnvEqual" {
		return os.Getenv(t.Key) == t.Value
	}
	if t.Operator == "EnvNotEqual" {
		return os.Getenv(t.Key) != t.Value
	}
	if t.Operator == "EnvNotEndsWith" {
		return !strings.HasSuffix(os.Getenv(t.Key), t.Value)
	}
	if t.Operator == "EnvEndsWith" {
		return strings.HasSuffix(os.Getenv(t.Key), t.Value)
	}
	if t.Operator == "EnvNotStartsWith" {
		return !strings.HasPrefix(os.Getenv(t.Key), t.Value)
	}
	if t.Operator == "EnvStartsWith" {
		return strings.HasPrefix(os.Getenv(t.Key), t.Value)
	}
	panic(fmt.Errorf("Unsupported operator %s ", t.Operator))
}

// IsActive evaluates the expression
func (t *Enabler) IsActive() bool {
	return t.Operator != ""
}

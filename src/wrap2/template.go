package main

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"go.uber.org/zap"
)

// Template holds information about processing
type Template struct {
	Source  string            `toml:"src"`
	Target  string            `toml:"dst"`
	Context map[string]string `toml:"data, omitempty"`
}

func secret(name string) string {
	src := path.Join("/etc/secrets", name)
	if _, err := os.Stat(src); os.IsNotExist(err) {
		log.Fatal("unable to locate", zap.String("secret", src), zap.Error(err))
	}

	data, err := ioutil.ReadFile(src)
	if err != nil {
		log.Fatal("unable to read", zap.String("secret", src), zap.Error(err))
	}
	return string(data)
}

func sha(value string) string {
	h := sha1.New()
	io.WriteString(h, value)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Process creates out file from src template
func (t *Template) Process() error {
	if t.Source != "" {
		if _, err := os.Stat(t.Source); os.IsNotExist(err) {
			log.Fatal("unable to open", zap.String("src", t.Source), zap.Error(err))
		}
	} else {
		return nil
	}

	data, err := ioutil.ReadFile(t.Source)
	if err != nil {
		return err
	}
	tmpl, err := template.New(t.Source).Funcs(template.FuncMap{
		"replace": strings.Replace,
		"lower":   strings.ToLower,
		"upper":   strings.ToUpper,
		"env":     os.Getenv,
		"k8s":     secret,
		"sha1":    sha,
	}).Parse(string(data))

	if err != nil {
		return err
	}

	if t.Target != "" {
		dest, err := os.Create(t.Target)
		if err != nil {
			log.Fatal("unable to create", zap.String("dest", t.Target), zap.Error(err))
		}
		defer dest.Close()
		if err := tmpl.Execute(dest, &t.Context); err != nil {
			log.Fatal("template error", zap.Error(err))
		}
	} else {
		return errors.New("`dst` is empty")
	}

	return nil
}

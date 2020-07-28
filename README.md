# wrap2

`wrap2` is process and configuration manager tailored for Docker with Kubernetes 
environment.

It supports:

- Template generation from environ and secrets
- Managing multiple processes from single declaration file
- PID1 handling of system events
- Cron jobs


## Template

Format is based of Golang default template parser [see](https://golang.org/pkg/text/template/):

Extra functions:

 - `env` > gets value from environment
 - `k8s` > load value from secret Kubernetes volume mount

## CRON

Based of https://godoc.org/github.com/robfig/cron:

Predefined schedules:
```
Entry                  | Description                                | Equivalent To
-----                  | -----------                                | -------------
@yearly (or @annually) | Run once a year, midnight, Jan. 1st        | 0 0 0 1 1 *
@monthly               | Run once a month, midnight, first of month | 0 0 0 1 * *
@weekly                | Run once a week, midnight between Sat/Sun  | 0 0 0 * * 0
@daily (or @midnight)  | Run once a day, midnight                   | 0 0 0 * * *
@hourly                | Run once an hour, beginning of hour        | 0 0 * * * *
```

### Example:

```
ENV var example {{env "CUSTOM"}}
Secret var example {{k8s "secret"}}

{{if {{env "DEFINED_ENV_VARIABLE"}}}} 
    {{k8s "special_secret"}}
{{end}}

```

## Configuration

```toml
[pre_start]
  cmd = "/bin/bash /pre_start"
  user = "dz0ny"
[post_start]
  cmd = "/bin/bash /post_start"
  user = "dz0ny"

[[process]]
  cmd = "nginx -V -E"
  user = "www-data"
  [process.config]
    src = "source.tmpl"
    out = "target.tmpl"
    [process.config.data]
      domain = "test.tld"

[[process]]
  cmd = "php -v"
  user = "www-data"
  [process.config]
    src = "source.tmpl"
    out = "target.tmpl"
   [process.enabled]
     operator="EnvNotEqual"
     # EnvEqual > env variable matches value
     # EnvNotEqual > env variable does not match value

     # EnvNotEndsWith > env variable does not end with value
     # EnvEndsWith > env variable ends with value

     # EnvNotStartsWith > env variable does not start with value
     # EnvStartsWith > env variable starts with value
     key= "KIND"
     value="scaled"

[[process]]
  cmd = "true -v"

[[cron]]
  schedule = "@every 5s"
  [cron.job]
    cmd = "echo tick"
    user = "www-data"
```

## Usage

```shell
$ ./wrap2-Linux-x86_64 -help                                                                                05/09/18 -  1:24 PM
Usage of ./wrap2-Linux-x86_64:
  -config string
        Location of the init file (default "/provision/init.toml")
```
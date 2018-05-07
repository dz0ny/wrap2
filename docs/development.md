# Developing

These instructions will help you set up a development environment for working on the `wrap2` source code.

## Prerequisites

To compile and test `wrap2`, you will need:

- make
- git
- [Go 1.10 or later](https://golang.org/doc/install)

In most cases, install each prerequisite according to its instructions.

## Setup

After you have setup Golang runtime and SDK, you need to setup root of the project,
the GOPATH environment variable which must be in the root of the project.

```shell
$ git clone git@github.com:dz0ny/`wrap2`.git
$ export GOPATH=$(pwd)
$ export GOBIN="$GOPATH/bin"
$ export PATH="$GOPATH/bin:$PATH"
```

The `GOROOT` needs to be present in enviroment as instructed by Golang install docs.
You can also use [gobu](https://github.com/dz0ny/gobu) to bootstrap environment. In this case
environment and runtime is prepared according to above instructions automatically.

## Making changes

Start by creating new branch prefixed with `feature/short-name` or `docs/short-name` or `cleanup/short-name`, depending on the change you are working on.

Run `make ensure` to install project dependencies.

### Test your changes

Run `make tests` to ensure the project is developed with best practices in mind.

## Making a release

The version is injected into the app at compilation time, where version is same one as defined in Makefile. Actual compilation happens in CI when master branch is tagged with version.

1. Change `VERSION` in Makefile according to [Semver](https://semver.org/) rules.
2. Make a PR with version change adn wait for merge to master.
3. Run `make release` which will do all necessary steps.

After CI built the project, Debian packages are published to PackageCloud `ebn-stack`
repository, where they are available for installation via Debian APT on servers.

## Debian package

Template for debian package is stored in .packaging directory.

## Backward compatibility

`wrap2` maintains a strong commitment to backward compatibility. All of our changes to protocols and formats are backward compatible. No features, flags, or commands are removed or substantially modified (other than bug fixes).

We also try very hard to not change publicly accessible Go library definitions inside of the src/ directory of our source code, but it is not guaranteed to be backward compatible as we move the project forward.

For a quick summary of our backward compatibility guidelines for major releases:

- Command line commands, flags, and arguments MUST be backward compatible

Other changes are permissable according to [Semver](https://semver.org/).
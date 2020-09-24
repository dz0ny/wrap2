VERSION := 1.5.5
PKG := github.com/dz0ny/wrap2
COMMIT := $(shell git rev-parse HEAD)
BUILD_TIME := $(shell date -u +%FT%T)
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
CURRENT_TARGET = wrap2-$(shell uname -s)-$(shell uname -m)
TARGETS := Linux-amd64-x86_64

os = $(word 1, $(subst -, ,$@))
arch = $(word 3, $(subst -, ,$@))
goarch = $(word 2, $(subst -, ,$@))
goos = $(shell echo $(os) | tr A-Z a-z)
output = wrap2-$(os)-$(arch)
version_flags = -X $(PKG)/version.Version=$(VERSION) \
 -X $(PKG)/version.CommitHash=${COMMIT} \
 -X $(PKG)/version.Branch=${BRANCH} \
 -X $(PKG)/version.BuildTime=${BUILD_TIME}

define localbuild
	GO111MODULE=off go get -u $(1)
	GO111MODULE=off go build $(1)
	mkdir -p bin
	mv $(2) bin/$(2)
endef

.PHONY: $(TARGETS)
$(TARGETS):
	env GOOS=$(goos) GOARCH=$(goarch) go build -trimpath --ldflags '-s -w $(version_flags)' -o $(output) $(PKG)

#
# Build all defined targets
#
.PHONY: build
build: $(TARGETS)

.PHONY: ensure
ensure:
	go get $(MODULE)

bin/gocov:
	$(call localbuild,github.com/axw/gocov/gocov,gocov)

bin/golangci-lint:
	$(call localbuild,github.com/golangci/golangci-lint/cmd/golangci-lint,golangci-lint)

clean:
	rm -f $(PKG)
	rm -rf pkg
	rm -rf bin
	find src/* -maxdepth 0 ! -name '$(PKG)' -type d | xargs rm -rf
	rm -rf src/$(PKG)/vendor/

lint: bin/golangci-lint
	bin/golangci-lint run
	go fmt

test: lint cover
	go test -v -race

cover: bin/gocov
	bin/gocov test | bin/gocov report

all: ensure build test

run:
	./wrap2-Linux-x86_64 --config=init.toml --logger=log.sock --debug

release:
	git stash
	git fetch -p
	git checkout master
	git pull -r
	git tag v$(VERSION)
	git push origin v$(VERSION)
	git pull -r

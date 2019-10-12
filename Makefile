VERSION := 1.2.9
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

define ghupload
	bin/github-release upload \
		--user dz0ny \
		--repo $(PKG) \
		--tag "v$(VERSION)" \
		--name $(PKG)-$(1) \
		--file $(PKG)-$(1)
endef

.PHONY: $(TARGETS)
$(TARGETS):
	env GOOS=$(goos) GOARCH=$(goarch) go build --ldflags '-s -w $(version_flags)' -o $(output) $(PKG)

#
# Build all defined targets
#
.PHONY: build
build: $(TARGETS)


bin/github-release:
	go get -u github.com/aktau/github-release

bin/gocov:
	go get -u github.com/axw/gocov/gocov

bin/golangci-lint:
	go get -u github.com/golangci/golangci-lint/cmd/golangci-lint

bin/go-ls:
	go get github.com/laher/gols/cmd/go-ls

clean:
	rm -f $(PKG)
	rm -rf pkg
	rm -rf bin
	find src/* -maxdepth 0 ! -name '$(PKG)' -type d | xargs rm -rf
	rm -rf src/$(PKG)/vendor/

lint: bin/golangci-lint
	bin/golangci-lint run src/$(PKG)/...
	find src/$(PKG) -not -path "./src/$(PKG)/vendor/*" -name '*.go' | xargs gofmt -w -s

test: bin/go-ls lint cover
	go test -v -race $(shell go-ls $(PKG)/...)

cover: bin/gocov
	gocov test $(shell go-ls $(PKG)/...) | gocov report

upload: bin/github-release
	$(call ghupload,Linux-x86_64)

all: ensure build test


run:
	./wrap2-Linux-x86_64 --config=init.toml --logger=log.sock

release:
	git stash
	git fetch -p
	git checkout master
	git pull -r
	git tag v$(VERSION)
	git push origin v$(VERSION)
	git pull -r

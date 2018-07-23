VERSION := 0.1.3
PKG := wrap2
COMMIT := $(shell git rev-parse HEAD)
BUILD_TIME := $(shell date -u +%FT%T)
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
CURRENT_TARGET = $(PKG)-$(shell uname -s)-$(shell uname -m)
TARGETS := Linux-amd64-x86_64

os = $(word 1, $(subst -, ,$@))
arch = $(word 3, $(subst -, ,$@))
goarch = $(word 2, $(subst -, ,$@))
goos = $(shell echo $(os) | tr A-Z a-z)
output = $(PKG)-$(os)-$(arch)
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

#
# Install app for current system
#
install: build
	sudo mv $(CURRENT_TARGET) /usr/local/bin/$(PKG)

#
# Install locked dependecies
#
ensure: bin/dep
	cd src/$(PKG); dep ensure

#
# Update all locked dependecies
#
update: bin/dep
	cd src/$(PKG); dep ensure -update

bin/dep:
	go get -u github.com/golang/dep/cmd/dep

bin/github-release:
	go get -u github.com/aktau/github-release

bin/gocov:
	go get -u github.com/axw/gocov/gocov

bin/gometalinter:
	go get -u github.com/alecthomas/gometalinter
	bin/gometalinter --install --update

bin/go-ls:
	go get github.com/laher/gols/cmd/go-ls

clean:
	rm -f $(PKG)
	rm -rf pkg
	rm -rf bin
	find src/* -maxdepth 0 ! -name '$(PKG)' -type d | xargs rm -rf
	rm -rf src/$(PKG)/vendor/
	 
lint: bin/gometalinter
	bin/gometalinter --fast --disable=gotype --disable=gosimple --disable=ineffassign --disable=dupl --disable=gas --cyclo-over=30 --deadline=60s --exclude $(shell pwd)/src/$(PKG)/vendor src/$(PKG)/...
	find src/$(PKG) -not -path "./src/$(PKG)/vendor/*" -name '*.go' | xargs gofmt -w -s

test: bin/go-ls lint cover
	go test -v -race $(shell go-ls $(PKG)/...)

cover: bin/gocov
	gocov test $(shell go-ls $(PKG)/...) | gocov report

upload: bin/github-release
	$(call ghupload,Linux-x86_64)

all: ensure build test

release:
	git stash
	git fetch -p
	git checkout master
	git pull -r
	git tag v$(VERSION)
	git push origin v$(VERSION)
	git pull -r

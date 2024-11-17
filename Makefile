VERSION=v2.0.0
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOFUMPT=gofumpt
WSL=wsl
GOLINT=golangci-lint run
BUILDPLATFORM=linux/amd64
VERSION_MAJOR=$(shell echo $(VERSION) | cut -f1 -d.)
VERSION_MINOR=$(shell echo $(VERSION) | cut -f2 -d.)
GIT_COMMIT=$(shell git rev-parse HEAD)
GIT_DIRTY=$(shell test -n "`git status --porcelain`" && echo "+CHANGES" || true)
BUILD_DATE=$(shell date '+%Y-%m-%d-%H:%M:%S')
# Image name
GO_PACKAGE=fahy.xyz/livetrack
GO_REGISTRY := $(if ${REGISTRY},$(patsubst %/,%,${REGISTRY})/)

all: ensure package

ensure:
	env GOOS=linux $(GOCMD) mod download

clean:
	$(GOCLEAN)

fmt:
	$(GOFUMPT) -l -w .
	$(WSL) --fix ./...

lint:
	$(GOLINT) ./...

test:
	$(GOTEST) ./...

package-bot:
	docker buildx build -f ./Dockerfile \
		--platform $(BUILDPLATFORM) \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg GIT_DIRTY=$(GIT_DIRTY) \
		-t $(GO_REGISTRY)$(GO_PACKAGE)-bot:$(VERSION) \
		-t $(GO_REGISTRY)$(GO_PACKAGE)-bot:$(VERSION_MAJOR).$(VERSION_MINOR) \
		-t $(GO_REGISTRY)$(GO_PACKAGE)-bot:$(VERSION_MAJOR) \
		--load \
		--target bot \
		.

package-web: 
	docker buildx build -f ./Dockerfile \
		--platform $(BUILDPLATFORM) \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg GIT_COMMIT=${GIT_COMMIT} \
		--build-arg GIT_DIRTY=${GIT_DIRTY} \
		--build-arg GIT_BRANCH=${GIT_BRANCH} \
		--build-arg GIT_USER=${GIT_USER} \
		-t $(GO_REGISTRY)$(GO_PACKAGE)-web:$(VERSION) \
		-t $(GO_REGISTRY)$(GO_PACKAGE)-web:$(VERSION_MAJOR).$(VERSION_MINOR) \
		-t $(GO_REGISTRY)$(GO_PACKAGE)-web:$(VERSION_MAJOR) \
		--load \
		--target web \
		.

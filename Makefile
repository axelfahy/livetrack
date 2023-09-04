VERSION=v1.1.1
VERSION_FRONTEND=v0.0.1-rc7
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOLINT=golangci-lint run
BUILD_PLATFORM=linux/amd64
PACKAGE_PLATFORM=$(BUILD_PLATFORM)
VERSION_MAJOR=$(shell echo $(VERSION) | cut -f1 -d.)
VERSION_MINOR=$(shell echo $(VERSION) | cut -f2 -d.)
GIT_COMMIT=$(shell git rev-parse HEAD)
GIT_DIRTY=$(shell test -n "`git status --porcelain`" && echo "+CHANGES" || true)
BUILD_DATE=$(shell date '+%Y-%m-%d-%H:%M:%S')
# Image name
GO_PACKAGE=fahy.xyz/livetrack
FRONTEND_PACKAGE=fahy.xyz/livetrack-web

all: ensure package 

ensure:
	env GOOS=linux $(GOCMD) mod download

clean:
	$(GOCLEAN)

lint:
	$(GOLINT) ./...

package:
	docker buildx build -f ./Dockerfile \
		--platform $(BUILD_PLATFORM) \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg GIT_DIRTY=$(GIT_DIRTY) \
		-t $(GO_PACKAGE):$(VERSION) \
		-t $(GO_PACKAGE):$(VERSION_MAJOR).$(VERSION_MINOR) \
		-t $(GO_PACKAGE):$(VERSION_MAJOR) \
		--load \
		.

frontend:
	docker buildx build -f web/frontend/Dockerfile \
		-t $(FRONTEND_PACKAGE):$(VERSION_FRONTEND) \
		--load \
		web/frontend/

test:
	$(GOTEST) ./...

VERSION=v2.4.0-rc2
BUILDPLATFORM=linux/arm64
GIT_COMMIT=$(shell git rev-parse HEAD)
GIT_DIRTY=$(shell test -n "`git status --porcelain`" && echo "+CHANGES" || true)
BUILD_DATE=$(shell date '+%Y-%m-%d-%H:%M:%S')
# Image name
GO_PACKAGE=fahy.xyz/livetrack
GO_REGISTRY := $(if ${REGISTRY},$(patsubst %/,%,${REGISTRY})/)

all: ensure push-api push-bot push-fetcher push-web

ensure:
	env GOOS=linux go mod download

clean:
	go clean
	go mod tidy

fmt:
	gofumpt -l -w .
	wsl --fix ./...

lint:
	golangci-lint run -c .golangci.yaml ./...

test:
	go test -race ./...

package-api:
	docker buildx build -f ./Dockerfile \
		--platform $(BUILDPLATFORM) \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg GIT_DIRTY=$(GIT_DIRTY) \
		-t $(GO_REGISTRY)$(GO_PACKAGE)-api:$(VERSION) \
		--load \
		--target api \
		.

push-api: package-api
	docker push $(GO_REGISTRY)$(GO_PACKAGE)-api:$(VERSION)

package-bot:
	docker buildx build -f ./Dockerfile \
		--platform $(BUILDPLATFORM) \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg GIT_DIRTY=$(GIT_DIRTY) \
		-t $(GO_REGISTRY)$(GO_PACKAGE)-bot:$(VERSION) \
		--load \
		--target bot \
		.

push-bot: package-bot
	docker push $(GO_REGISTRY)$(GO_PACKAGE)-bot:$(VERSION)

package-fetcher:
	docker buildx build -f ./Dockerfile \
		--platform $(BUILDPLATFORM) \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg GIT_DIRTY=$(GIT_DIRTY) \
		-t $(GO_REGISTRY)$(GO_PACKAGE)-fetcher:$(VERSION) \
		--load \
		--target fetcher \
		.

push-fetcher: package-fetcher
	docker push $(GO_REGISTRY)$(GO_PACKAGE)-fetcher:$(VERSION)

package-sse:
	docker buildx build -f ./Dockerfile \
		--platform $(BUILDPLATFORM) \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg GIT_DIRTY=$(GIT_DIRTY) \
		-t $(GO_REGISTRY)$(GO_PACKAGE)-sse:$(VERSION) \
		--load \
		--target sse \
		.

push-sse: package-sse
	docker push $(GO_REGISTRY)$(GO_PACKAGE)-sse:$(VERSION)

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
		--load \
		--target web \
		.

push-web: package-web
	docker push $(GO_REGISTRY)$(GO_PACKAGE)-web:$(VERSION)


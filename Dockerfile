FROM --platform=$BUILDPLATFORM golang:alpine AS builder

ARG TARGETOS
ARG TARGETARCH
ARG GIT_COMMIT
ARG GIT_DIRTY
ARG BUILD_DATE
ARG VERSION

COPY . /src

WORKDIR /src

FROM builder AS builder-api
RUN env GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 \
    go build -trimpath -o livetrack-api \
    -ldflags "-w -s \
              -X github.com/prometheus/common/version.Version=${VERSION} \
              -X github.com/prometheus/common/version.Revision=${GIT_COMMIT}${GIT_DIRTY} \
              -X github.com/prometheus/common/version.Branch=${GIT_BRANCH} \
              -X github.com/prometheus/common/version.BuildUser=${GIT_USER} \
              -X github.com/prometheus/common/version.BuildDate=${BUILD_DATE}" \
    ./cmd/api

FROM builder AS builder-bot
RUN env GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 \
    go build -trimpath -o livetrack-bot \
    -ldflags "-w -s \
              -X github.com/prometheus/common/version.Version=${VERSION} \
              -X github.com/prometheus/common/version.Revision=${GIT_COMMIT}${GIT_DIRTY} \
              -X github.com/prometheus/common/version.Branch=${GIT_BRANCH} \
              -X github.com/prometheus/common/version.BuildUser=${GIT_USER} \
              -X github.com/prometheus/common/version.BuildDate=${BUILD_DATE}" \
    ./cmd/bot

FROM builder AS builder-fetcher
RUN env GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 \
    go build -trimpath -o livetrack-fetcher \
    -ldflags "-w -s \
              -X github.com/prometheus/common/version.Version=${VERSION} \
              -X github.com/prometheus/common/version.Revision=${GIT_COMMIT}${GIT_DIRTY} \
              -X github.com/prometheus/common/version.Branch=${GIT_BRANCH} \
              -X github.com/prometheus/common/version.BuildUser=${GIT_USER} \
              -X github.com/prometheus/common/version.BuildDate=${BUILD_DATE}" \
    ./cmd/fetcher

FROM builder AS builder-sse
RUN env GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 \
    go build -trimpath -o livetrack-sse \
    -ldflags "-w -s \
              -X github.com/prometheus/common/version.Version=${VERSION} \
              -X github.com/prometheus/common/version.Revision=${GIT_COMMIT}${GIT_DIRTY} \
              -X github.com/prometheus/common/version.Branch=${GIT_BRANCH} \
              -X github.com/prometheus/common/version.BuildUser=${GIT_USER} \
              -X github.com/prometheus/common/version.BuildDate=${BUILD_DATE}" \
    ./cmd/sse

FROM builder AS builder-web
RUN env GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 \
    go build -trimpath -o livetrack-web \
    -ldflags "-w -s \
              -X github.com/prometheus/common/version.Version=${VERSION} \
              -X github.com/prometheus/common/version.Revision=${GIT_COMMIT}${GIT_DIRTY} \
              -X github.com/prometheus/common/version.Branch=${GIT_BRANCH} \
              -X github.com/prometheus/common/version.BuildUser=${GIT_USER} \
              -X github.com/prometheus/common/version.BuildDate=${BUILD_DATE}" \
    ./cmd/web

FROM --platform=$BUILDPLATFORM alpine:3.21 AS api
COPY --from=builder-api /src/livetrack-api /livetrack-api
CMD ["/livetrack-api"]

FROM --platform=$BUILDPLATFORM alpine:3.21 AS bot
COPY --from=builder-bot /src/livetrack-bot /livetrack-bot
CMD ["/livetrack-bot"]

FROM --platform=$BUILDPLATFORM alpine:3.21 AS fetcher
COPY --from=builder-fetcher /src/livetrack-fetcher /livetrack-fetcher
CMD ["/livetrack-fetcher"]

FROM --platform=$BUILDPLATFORM alpine:3.21 AS sse
COPY --from=builder-sse /src/livetrack-sse /livetrack-sse
CMD ["/livetrack-sse"]

FROM --platform=$BUILDPLATFORM alpine:3.21 AS web
COPY --from=builder-web /src/livetrack-web /livetrack-web
CMD ["/livetrack-web"]

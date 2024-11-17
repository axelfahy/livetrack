FROM --platform=$BUILDPLATFORM golang:alpine AS builder

ARG TARGETOS
ARG TARGETARCH
ARG GIT_COMMIT
ARG GIT_DIRTY
ARG BUILD_DATE
ARG VERSION

COPY . /src

WORKDIR /src

RUN env GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 \
    go build -o livetrack-api \
    -ldflags "-w -s \
              -X github.com/prometheus/common/version.Version=${VERSION} \
              -X github.com/prometheus/common/version.Revision=${GIT_COMMIT}${GIT_DIRTY} \
              -X github.com/prometheus/common/version.Branch=${GIT_BRANCH} \
              -X github.com/prometheus/common/version.BuildUser=${GIT_USER} \
              -X github.com/prometheus/common/version.BuildDate=${BUILD_DATE}" \
    ./cmd/api

RUN env GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 \
    go build -o livetrack-bot \
    -ldflags "-w -s \
              -X github.com/prometheus/common/version.Version=${VERSION} \
              -X github.com/prometheus/common/version.Revision=${GIT_COMMIT}${GIT_DIRTY} \
              -X github.com/prometheus/common/version.Branch=${GIT_BRANCH} \
              -X github.com/prometheus/common/version.BuildUser=${GIT_USER} \
              -X github.com/prometheus/common/version.BuildDate=${BUILD_DATE}" \
    ./cmd/bot

RUN env GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 \
    go build -o livetrack-fetcher \
    -ldflags "-w -s \
              -X github.com/prometheus/common/version.Version=${VERSION} \
              -X github.com/prometheus/common/version.Revision=${GIT_COMMIT}${GIT_DIRTY} \
              -X github.com/prometheus/common/version.Branch=${GIT_BRANCH} \
              -X github.com/prometheus/common/version.BuildUser=${GIT_USER} \
              -X github.com/prometheus/common/version.BuildDate=${BUILD_DATE}" \
    ./cmd/fetcher

RUN env GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 \
    go build -o livetrack-web \
    -ldflags "-w -s \
              -X github.com/prometheus/common/version.Version=${VERSION} \
              -X github.com/prometheus/common/version.Revision=${GIT_COMMIT}${GIT_DIRTY} \
              -X github.com/prometheus/common/version.Branch=${GIT_BRANCH} \
              -X github.com/prometheus/common/version.BuildUser=${GIT_USER} \
              -X github.com/prometheus/common/version.BuildDate=${BUILD_DATE}" \
    ./cmd/web

FROM --platform=$BUILDPLATFORM alpine:3.20 AS api
COPY --from=builder /src/livetrack-api /livetrack-api
CMD ["/livetrack-api"]

FROM --platform=$BUILDPLATFORM alpine:3.20 AS bot
COPY --from=builder /src/livetrack-bot /livetrack-bot
CMD ["/livetrack-bot"]

FROM --platform=$BUILDPLATFORM alpine:3.20 AS fetcher
COPY --from=builder /src/livetrack-fetcher /livetrack-fetcher
CMD ["/livetrack-fetcher"]

FROM --platform=$BUILDPLATFORM alpine:3.20 AS web
COPY --from=builder /src/livetrack-web /livetrack-web
CMD ["/livetrack-web"]
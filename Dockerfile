FROM --platform=$BUILDPLATFORM golang:alpine AS builder

ARG TARGETOS
ARG TARGETARCH
ARG GIT_COMMIT
ARG GIT_DIRTY
ARG VERSION
ARG BUILD_DATE

COPY . /src

WORKDIR /src

RUN env GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 go mod download && \
    env GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 \
    go build -o livetrack \
    -ldflags "-X github.com/sqooba/go-common/version.GitCommit=${GIT_COMMIT}${GIT_DIRTY} \
              -X github.com/sqooba/go-common/version.BuildDate=${BUILD_DATE} \
              -X github.com/sqooba/go-common/version.Version=${VERSION}" \
    ./main.go

FROM --platform=$BUILDPLATFORM alpine:3.16

COPY --from=builder /src/livetrack /livetrack

CMD ["/livetrack"]

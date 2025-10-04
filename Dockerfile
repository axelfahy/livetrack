FROM alpine:3.22 AS api
ARG TARGETPLATFORM
COPY  $TARGETPLATFORM/livetrack-api /usr/bin/livetrack-api
CMD ["/usr/bin/livetrack-api"]

FROM alpine:3.22 AS bot
ARG TARGETPLATFORM
COPY  $TARGETPLATFORM/livetrack-bot /usr/bin/livetrack-bot
CMD ["/usr/bin/livetrack-bot"]

FROM alpine:3.22 AS fetcher
ARG TARGETPLATFORM
COPY  $TARGETPLATFORM/livetrack-fetcher /usr/bin/livetrack-fetcher
CMD ["/usr/bin/livetrack-fetcher"]

FROM alpine:3.22 AS sse
ARG TARGETPLATFORM
COPY  $TARGETPLATFORM/livetrack-sse /usr/bin/livetrack-sse
CMD ["/usr/bin/livetrack-sse"]

FROM alpine:3.22 AS web
ARG TARGETPLATFORM
COPY  $TARGETPLATFORM/livetrack-web /usr/bin/livetrack-web
CMD ["/usr/bin/livetrack-web"]

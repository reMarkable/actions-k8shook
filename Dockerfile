FROM  golang:1.25.3-trixie AS builder

ARG BUILDPLATFORM
ARG TARGETOS="linux"
ARG TARGETARCH="amd64"
ARG VERSION="docker-dev"

WORKDIR /app/
ADD . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-X main.version=${VERSION} -w -s" -o hook hook.go

FROM ghcr.io/myoung34/docker-github-actions-runner:ubuntu-noble
RUN mkdir /hook
ADD hook.sh /hook/hook.sh
COPY --from=builder /app/hook /hook/hook
RUN chmod 755 /hook/*
# ENV DEBUG_HOOK=1

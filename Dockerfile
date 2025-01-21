FROM ghcr.io/openfaas/license-check:0.4.2 AS license-check

FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.23 AS build

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

ENV CGO_ENABLED=0
ENV GO111MODULE=on

COPY --from=license-check /license-check /usr/bin/

WORKDIR /go/src/github.com/openfaas/cron-connector
COPY . .

RUN license-check -path /go/src/github.com/openfaas/cron-connector/ --verbose=false "Alex Ellis" "OpenFaaS Author(s)"
RUN gofmt -l -d $(find . -type f -name '*.go' -not -path "./vendor/*")
RUN CGO_ENABLED=${CGO_ENABLED} GOOS=${TARGETOS} GOARCH=${TARGETARCH} go test -v ./...

RUN VERSION=$(git describe --all --exact-match `git rev-parse HEAD` | grep tags | sed 's/tags\///') \
    && GIT_COMMIT=$(git rev-list -1 HEAD) \
    && GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=${CGO_ENABLED} go build \
        --ldflags "-s -w \
        -X github.com/openfaas/cron-connector/version.GitCommit=${GIT_COMMIT}\
        -X github.com/openfaas/cron-connector/version.Version=${VERSION}" \
        -o cron-connector .

FROM --platform=${TARGETPLATFORM:-linux/amd64} alpine:3.21.2 AS ship
LABEL org.label-schema.license="MIT" \
      org.label-schema.vcs-url="https://github.com/openfaas/cron-connector" \
      org.label-schema.vcs-type="Git" \
      org.label-schema.name="openfaas/cron-connector" \
      org.label-schema.vendor="openfaas" \
      org.label-schema.docker.schema-version="1.0"

RUN apk --no-cache add \
    ca-certificates

RUN addgroup -S app \
    && adduser -S -g app app

WORKDIR /home/app

ENV http_proxy      ""
ENV https_proxy     ""

COPY --from=build /go/src/github.com/openfaas/cron-connector/cron-connector    .
RUN chown -R app:app ./

USER app

CMD ["./cron-connector"]

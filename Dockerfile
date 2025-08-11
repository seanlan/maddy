FROM golang:1.23-alpine AS build-env

ARG ADDITIONAL_BUILD_TAGS=""

RUN set -ex && \
    apk upgrade --no-cache --available && \
    apk add --no-cache build-base

WORKDIR /mailchat

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN mkdir -p /pkg/data && \
    cp maddy.conf.docker /pkg/data/mailchat.conf && \
    ./build.sh --builddir /tmp --destdir /pkg/ --tags "docker ${ADDITIONAL_BUILD_TAGS}" build install

FROM alpine:3.21.2
LABEL maintainer="fox.cpp@disroot.org"
LABEL org.opencontainers.image.source=https://github.com/dsoftgames/MailChat

RUN set -ex && \
    apk upgrade --no-cache --available && \
    apk --no-cache add ca-certificates
COPY --from=build-env /pkg/data/mailchat.conf /data/mailchat.conf
COPY --from=build-env /pkg/usr/local/bin/mailchat /bin/

EXPOSE 25 143 993 587 465
VOLUME ["/data"]
ENTRYPOINT ["/bin/mailchat", "-config", "/data/mailchat.conf"]
CMD ["run"]

#!/bin/sh

DEFAULT_TAGS=static,netgo

###
# -ldflags='-w -s -extldflags "-static"'
# seems to cause problems in tandem with -tags static,netgo
# so we're leaving it out for now, given upstream (confluent-kafka-go) docs
# recommend using -tags static instead of the -ldflags layout
###
if which apk > /dev/null 2>&1; then
  echo -e 'https://mirrors.aliyun.com/alpine/v3.19/main/\nhttps://mirrors.aliyun.com/alpine/v3.19/community/' > /etc/apk/repositories

	apk add --no-cache gcc musl-dev
	go build -tags "musl,${DEFAULT_TAGS}" -mod=vendor -o "$1-musl" ./...
else
	go build -buildvcs=false -tags "${DEFAULT_TAGS}" -o "$1-libc"  ./...
fi

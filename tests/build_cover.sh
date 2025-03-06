#!/bin/sh
if [ -z "$GO" ]; then
	GO=go
fi
exec $GO test -tags 'cover_main debugflags' -coverpkg 'mailcoin,mailcoin/pkg/...,mailcoin/internal/...' -cover -covermode atomic -c cover_test.go -o maddy.cover

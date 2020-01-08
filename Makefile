.PHONY : build test clean

LAST_COMMIT := $(shell git rev-parse --short HEAD)
LAST_COMMIT_DATE := $(shell git show -s --format=%ci ${LAST_COMMIT})
VERSION := $(shell git describe --abbrev=1)
BUILDSTR := ${VERSION} (build "\\\#"${LAST_COMMIT} $(shell date '+%Y-%m-%d %H:%M:%S'))

BIN := pfxsigner

build:
	go build -o ${BIN} -ldflags="-s -w -X 'main.buildString=${BUILDSTR}'"

test:
	go test ./...

clean:
	go clean
	- rm -f ${BIN}

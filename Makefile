.PHONY: default etools all clean fmt

GOBIN = $(shell pwd)/build/bin
TAG ?= latest
GOFILES_NOVENDOR := $(shell go list -f "{{.Dir}}" ./...)

COMMIT_SHA1 := $(shell git rev-parse HEAD)
AppName := etools

default: etools

all: etools

BUILD_FLAGS = -tags netgo -ldflags "\
	-X github.com/xueqianLu/ethtools/versions.AppName=${AppName} \
	-X 'github.com/xueqianLu/ethtools/versions.BuildTime=`date`' \
	-X github.com/xueqianLu/ethtools/versions.CommitSha1=${COMMIT_SHA1}  \
	-X 'github.com/xueqianLu/ethtools/versions.GoVersion=`go version`' \
	-X 'github.com/xueqianLu/ethtools/versions.GitBranch=`git symbolic-ref --short -q HEAD`' \
	"

docker:
	docker build -t etools:${TAG} .

etools:
	go build $(BUILD_FLAGS) -o=${GOBIN}/$@ -gcflags "all=-N -l" ./
	@echo "Done building."

clean:
	rm -fr build/*

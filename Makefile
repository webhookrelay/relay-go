JOBDATE		?= $(shell date -u +%Y-%m-%dT%H%M%SZ)
GIT_REVISION	= $(shell git rev-parse --short HEAD)
VERSION		?= $(shell git describe --tags --abbrev=0)

LDFLAGS		+= -s -w
LDFLAGS		+= -X github.com/webhookrelay/relay-go/version.Version=$(VERSION)
LDFLAGS		+= -X github.com/webhookrelay/relay-go/version.Revision=$(GIT_REVISION)
LDFLAGS		+= -X github.com/webhookrelay/relay-go/version.BuildDate=$(JOBDATE)

install:
	cd cmd/relayd && go install -ldflags "$(LDFLAGS)"

test:
	go test -v -failfast `go list ./... | egrep -v /tests/`

test-pretty:
	go get github.com/mfridman/tparse	
	go test -json  -v `go list ./... | egrep -v /tests/` -cover | tparse -all -smallscreen

gen-json-types:
	easyjson -all pkg/types/ws_types.go
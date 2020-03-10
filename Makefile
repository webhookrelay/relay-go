JOBDATE		?= $(shell date -u +%Y-%m-%dT%H%M%SZ)
GIT_REVISION	= $(shell git rev-parse --short HEAD)
VERSION		?= $(shell git describe --tags --abbrev=0)

LDFLAGS		+= -s -w
LDFLAGS		+= -X github.com/webhookrelay/relay-go/version.Version=$(VERSION)
LDFLAGS		+= -X github.com/webhookrelay/relay-go/version.Revision=$(GIT_REVISION)
LDFLAGS		+= -X github.com/webhookrelay/relay-go/version.BuildDate=$(JOBDATE)

install:
	cd cmd/relayd && go install -ldflags "$(LDFLAGS)"

release:
	cd cmd/relayd && env GOARCH=amd64 GOOS=linux go build -ldflags "$(LDFLAGS)" -o release/relayd
	cd cmd/relayd && env GOARCH=amd64 GOOS=windows go build -ldflags "$(LDFLAGS)" -o release/relayd.exe

test:
	go test -v -failfast `go list ./... | egrep -v /tests/`

e2e:
	go get github.com/mfridman/tparse
	cd tests/forward && go test

test-pretty:
	go get github.com/mfridman/tparse
	go test -json  -v `go list ./... | egrep -v /tests/` -cover | tparse -all -smallscreen

gen-json-types:
	go get -u github.com/mailru/easyjson/...
	easyjson -all pkg/types/ws_types.go
	easyjson -all pkg/types/request_log.go

run: install
	relayd forward
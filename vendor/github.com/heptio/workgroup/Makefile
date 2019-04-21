PKGS := $(shell go list ./...)

test: install
	go test ./...

check: test vet staticcheck unused
	@echo Checking code is gofmted
	@bash -c 'if [ -n "$(gofmt -s -l .)" ]; then echo "Go code is not formatted:"; gofmt -s -d -e .; exit 1;fi'

install:
	go install -v ./...

vet: | test
	go vet ./...

staticcheck:
	@go get honnef.co/go/tools/cmd/staticcheck
	staticcheck $(PKGS)

unused:
	@go get honnef.co/go/tools/cmd/unused
	unused $(PKGS)


build:
	@go generate ./...
	@go build ./...

lint:
	@scripts/run_golangci.sh

fmt:
	@scripts/run_gofmt.sh

include Makefile.common.mk

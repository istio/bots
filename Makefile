
build:
	@go generate ./...
	@go build ./...

lint:
	# These PATH hacks are temporary until prow properly sets its paths
	@PATH=${PATH}:${GOPATH}/bin scripts/check_license.sh
	@PATH=${PATH}:${GOPATH}/bin scripts/run_golangci.sh

fmt:
	@scripts/run_gofmt.sh

include Makefile.common.mk

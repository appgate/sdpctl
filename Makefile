BIN_NAME=appgatectl
GOFMT_FILES?=$$(find . -name '*.go' | grep -v vendor)


build:
	go build -o $(BIN_NAME)

fmtcheck:
	@sh -c "'$(CURDIR)/scripts/gofmtcheck.sh'"

fmt:
	gofmt -w $(GOFMT_FILES)

test:
	go test ./...

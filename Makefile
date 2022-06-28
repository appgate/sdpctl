BIN_NAME=sdpctl
GOFMT_FILES?=$$(find . -name '*.go' | grep -v vendor)
DESTDIR :=
prefix  := /usr/local
bindir  := ${prefix}/bin
commit=$$(git rev-parse HEAD)
commitPath=github.com/appgate/sdpctl/cmd.commit=${commit}

.PHONY: build
build:
	CGO_ENABLED=0 go build -o build/$(BIN_NAME) -ldflags="-X '${commitPath}'"

.PHONY: deps
deps:
	mkdir -p build
	go run main.go completion bash > build/bash_completion
	go run main.go generate man

snapshot: clean
	goreleaser release --snapshot --rm-dist

fmtcheck:
	@sh -c "'$(CURDIR)/scripts/gofmtcheck.sh'"

fmt:
	gofmt -w $(GOFMT_FILES)

# Run go test twice, since -race don't catch all edge cases
test:
	go test ./... -count 1 -timeout 30s
	go test ./... -race -covermode=atomic -coverprofile=cover.out -timeout 60s

cover: test
	go tool cover -func cover.out

clean:
	rm -rf build dist cover.out

.PHONY: install
install: build
	install -d ${DESTDIR}${bindir}
	install -m755 build/$(BIN_NAME) ${DESTDIR}${bindir}/


BIN_NAME=appgatectl
GOFMT_FILES?=$$(find . -name '*.go' | grep -v vendor)
DESTDIR :=
prefix  := /usr/local
bindir  := ${prefix}/bin

build:
	go build -o build/$(BIN_NAME)

fmtcheck:
	@sh -c "'$(CURDIR)/scripts/gofmtcheck.sh'"

fmt:
	gofmt -w $(GOFMT_FILES)

test:
	go test ./... -race -coverpkg=./cmd/appliance/...,./cmd/configure/... -covermode=atomic -coverprofile=cover.out

cover: test
	go tool cover -func cover.out

clean:
	rm -rf build dist cover.out


.PHONY: install
install: build
	install -d ${DESTDIR}${bindir}
	install -m755 build/$(BIN_NAME) ${DESTDIR}${bindir}/

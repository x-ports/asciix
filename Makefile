BINARY := asciix
PREFIX ?= $(HOME)/.local/bin
GO ?= go

.PHONY: build install uninstall fmt vet check clean

build:
	$(GO) build -o $(BINARY) .

install: build
	mkdir -p $(PREFIX)
	install -m 0755 $(BINARY) $(PREFIX)/$(BINARY)

uninstall:
	rm -f $(PREFIX)/$(BINARY)

fmt:
	gofmt -w .

vet:
	$(GO) vet ./...

check: vet
	test -z "$$(gofmt -l .)" || { echo "run 'make fmt'"; exit 1; }
	$(GO) build ./...

clean:
	rm -f $(BINARY)
	rm -rf dist

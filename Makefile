BINARY := gofetch
PREFIX ?= /usr/local
BINDIR := $(PREFIX)/bin

export CGO_ENABLED := 1

.PHONY: all build run vet fmt install uninstall clean

all: build

build:
	go build -o $(BINARY) .

run: build
	./$(BINARY)

vet:
	go vet ./...

fmt:
	gofmt -w *.go

install: build
	install -d $(DESTDIR)$(BINDIR)
	install -m 755 $(BINARY) $(DESTDIR)$(BINDIR)/$(BINARY)

uninstall:
	rm -f $(DESTDIR)$(BINDIR)/$(BINARY)

clean:
	rm -f $(BINARY)
	go clean

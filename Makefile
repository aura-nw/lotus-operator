GITCOMMIT := $(shell git rev-parse HEAD)
GITDATE := $(shell git show -s --format='%ct')

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

build:
	env GO111MODULE=on go build -v $(LDFLAGS) ./cmd/operator

test:
	go test -v ./...

lint:
	golangci-lint run ./...

.PHONY: \
	build \
	test \
	lint
MOCK_TIKV_PKG := github.com/tikv/mock-tikv

PACKAGES := go list ./...
PACKAGE_DIRECTORIES := $(PACKAGES) | sed 's|github.com/tikv/mock-tikv/||'
GOCHECKER := awk '{ print } END { if (NR > 0) { exit 1 } }'
RETOOL := ./scripts/retool

GOVER_MAJOR := $(shell go version | sed -E -e "s/.*go([0-9]+)[.]([0-9]+).*/\1/")
GOVER_MINOR := $(shell go version | sed -E -e "s/.*go([0-9]+)[.]([0-9]+).*/\2/")
GO111 := $(shell [ $(GOVER_MAJOR) -gt 1 ] || [ $(GOVER_MAJOR) -eq 1 ] && [ $(GOVER_MINOR) -ge 11 ]; echo $$?)
ifeq ($(GO111), 1)
$(error "go below 1.11 does not support modules")
endif

default: build

all: dev

dev: build check

build: export GO111MODULE=on
build:
ifeq ("$(WITH_RACE)", "1")
	CGO_ENABLED=1 go build -race -ldflags '$(LDFLAGS)' -o bin/mock-tikv cmd/mock-tikv/main.go
else
	CGO_ENABLED=0 go build -ldflags '$(LDFLAGS)' -o bin/mock-tikv cmd/mock-tikv/main.go
endif

# These need to be fixed before they can be ran regularly
check-fail:
	CGO_ENABLED=0 ./scripts/retool do gometalinter.v2 --disable-all \
	  --enable errcheck \
	  $$($(PACKAGE_DIRECTORIES))
	CGO_ENABLED=0 ./scripts/retool do gosec $$($(PACKAGE_DIRECTORIES))

check-all: static lint tidy
	@echo "checking"

retool-setup: export GO111MODULE=off
retool-setup:
	@which retool >/dev/null 2>&1 || go get github.com/twitchtv/retool
	@./scripts/retool sync

check: retool-setup check-all

static:
	@ # Not running vet and fmt through metalinter becauase it ends up looking at vendor
	gofmt -s -l $$($(PACKAGE_DIRECTORIES)) 2>&1 | $(GOCHECKER)
	./scripts/retool do govet --shadow $$($(PACKAGE_DIRECTORIES)) 2>&1 | $(GOCHECKER)

	CGO_ENABLED=0 ./scripts/retool do gometalinter.v2 --disable-all --deadline 120s \
	  --enable misspell \
	  --enable megacheck \
	  --enable ineffassign \
	  $$($(PACKAGE_DIRECTORIES))

lint:
	@echo "linting"
	CGO_ENABLED=0 ./scripts/retool do revive -formatter friendly -config revive.toml $$($(PACKAGES))

tidy:
	@echo "go mod tidy"
	GO111MODULE=on go mod tidy
	git diff --quiet

.PHONY: all tidy

-include eng/Makefile

.DEFAULT_GOAL = build

.PHONY: \
	-install-pastiche \

BUILD_VERSION=$(shell git rev-parse --short HEAD)
GO_LDFLAGS=-X 'github.com/Carbonfrost/pastiche/pkg/internal/build.Version=$(BUILD_VERSION)'

install: -install-pastiche

-install-pastiche: --install-pastiche

--install-%: build -check-env-PREFIX -check-env-_GO_OUTPUT_DIR Makefile
	$(Q) eng/install "${_GO_OUTPUT_DIR}/$*" $(PREFIX)/bin

test:
	$(Q) ginkgo ./...

lint:
	$(Q) go run honnef.co/go/tools/cmd/staticcheck -checks 'all,-ST*' $(shell go list ./...)

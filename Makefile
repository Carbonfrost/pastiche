# Copyright 2022, 2025 The Pastiche Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.
-include eng/Makefile

.DEFAULT_GOAL = build

.PHONY: \
	-install-pastiche \

BUILD_VERSION=$(shell git rev-parse --short HEAD)
GO_LDFLAGS=

install: -install-pastiche

-install-pastiche: --install-pastiche

--install-%: build -check-env-PREFIX -check-env-_GO_OUTPUT_DIR Makefile
	$(Q) eng/install "${_GO_OUTPUT_DIR}/$*" $(PREFIX)/bin

test:
	$(Q) ginkgo ./...

lint:
	$(Q) go vet ./... 2>&1 || true
	$(Q) go tool gocritic check ./... 2>&1 || true
	$(Q) go tool revive ./... 2>&1 || true
	$(Q) go tool staticcheck -checks 'all,-ST*' $(shell go list ./...) 2>&1 || true

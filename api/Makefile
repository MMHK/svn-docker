GO ?= go
SWAG_VERSION ?= v1.16.6

# This repository is a git repo nested inside an SVN working copy, so the Go
# tool refuses to stamp VCS metadata ("multiple VCS detected"). CI checkouts
# only contain git, but keeping the flag here makes every target runnable
# locally as well.
GOFLAGS ?= -buildvcs=false
export GOFLAGS

.PHONY: build test vet layers swag run check clean

build:
	$(GO) build ./...

test: vet layers
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

# MVP boundary from plan section 7.2 check 14: the presenter layer (and every
# package it pulls in) must not import net/http.
layers:
	@deps="$$($(GO) list -deps ./pkg/presenter)"; \
	if printf '%s\n' "$$deps" | grep -qx 'net/http'; then \
		echo "ERROR: pkg/presenter must not depend on net/http"; \
		exit 1; \
	fi

swag:
	$(GO) run github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION) init \
		-g cmd/svnadmin-api/main.go \
		-o pkg/docs \
		--parseDependency \
		--parseDepth 3

run:
	$(GO) run ./cmd/svnadmin-api serve

check:
	$(GO) run ./cmd/svnadmin-api check

clean:
	$(GO) clean ./...

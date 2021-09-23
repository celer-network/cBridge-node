VERSION := $(shell echo $(shell git describe --tags) | sed 's/^v//')
COMMIT := $(shell git log -1 --format='%H')

ldflags = -X main.version=$(VERSION) \
		  -X main.commit=$(COMMIT)

BUILD_FLAGS := -ldflags '$(ldflags)' -o cbridge-node

go.sum: go.mod
	@echo "--> Ensure dependencies have not been modified"
	GO111MODULE=on go mod verify

.PHONY: build
build: go.sum
	mkdir -p build && cd build && \
	go build $(BUILD_FLAGS) ../server/main

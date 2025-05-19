
BINARY_NAME=idefix-server
OUTPUT_DIR=./../dist/server
SRC=./


GOBIN := $(shell go env GOBIN)
ifeq ($(strip $(GOBIN)),)
GOBIN := $(shell go env GOPATH)/bin
endif
AIR := $(GOBIN)/air

$(AIR):
	@echo "→ installing Air …"
	go install github.com/cosmtrek/air@latest

all: build_arm build_arm64 build_linux

build_arm:
	GOOS=linux GOARCH=arm go build -o "$(OUTPUT_DIR)/arm/$(BINARY_NAME)" $(SRC)

build_arm64:
	GOOS=linux GOARCH=arm64 go build -o "$(OUTPUT_DIR)/arm64/$(BINARY_NAME)" $(SRC)

build_linux:
	GOOS=linux GOARCH=amd64 go build -o "$(OUTPUT_DIR)/amd64/$(BINARY_NAME)" $(SRC)

build_and_copy:
	make build_arm64
	cp $(OUTPUT_DIR)/arm64/$(BINARY_NAME) $(HOME)/router/opt/share/idefix/idefix-server

.PHONY: dev
dev:                                       
	@command -v $(AIR) >/dev/null 2>&1 || { \
		echo '→ installing Air …'; \
		go install github.com/air-verse/air@latest; \
	}
	$(AIR) -c .air.toml

clean:
	rm -f $(BINARY_NAME)_*
.PHONY: all test build-cli build-frontend build-windows clean

VERSION ?= 1.0.0
BUILD_DIR = bin

all: test build-cli build-windows

test:
	go test -v ./...

build-frontend:
	cd frontend && npm install --no-audit --no-fund && npm run build

build-cli:
	mkdir -p $(BUILD_DIR)
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o $(BUILD_DIR)/mediaflow ./cmd/mediaflow

build-windows: build-frontend
	mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -H windowsgui -X main.version=$(VERSION)" -o $(BUILD_DIR)/MediaFlow.exe ./cmd/desktop

clean:
	rm -rf $(BUILD_DIR) frontend/dist

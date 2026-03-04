default: build

build: lint
    go build ./...

lint:
    golangci-lint run ./...

fix:
    golangci-lint run --fix ./...

test:
    go test ./...

package:
    goreleaser release --snapshot --clean

test-packages: package
    docker compose build

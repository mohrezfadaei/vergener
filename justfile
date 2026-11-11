# Local development and container-image tasks.
set positional-arguments := true

default:
    @just --list

build:
    go build -o vergener ./cmd/vergener

test:
    go test ./...

test-race:
    go test -race ./...

vet:
    go vet ./...

check: test test-race vet

validate:
    go run ./cmd/vergener validate --config Vergener.yml

# Pass an explicit registry tag to build an image for publishing.
docker-build image="vergener:local":
    docker build --tag "$1" .

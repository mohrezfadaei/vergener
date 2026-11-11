# syntax=docker/dockerfile:1
FROM golang:1.27.1-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/vergener ./cmd/vergener

FROM alpine:3.23
RUN apk add --no-cache git ca-certificates
COPY --from=build /out/vergener /usr/local/bin/vergener

# No ENTRYPOINT: GitLab Runner must be able to start its job shell.
# The job's checkout and Vergener.yml are supplied at runtime, not in the image.

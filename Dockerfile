FROM golang:1.26.5-bookworm AS e2e

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV MCM_E2E_IN_DOCKER=1

RUN go test -v ./e2e

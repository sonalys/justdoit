FROM golang:1.22 as builder

RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.59.1
RUN go install golang.org/x/tools/cmd/deadcode@v0.23.0
RUN go install github.com/daveshanley/vacuum@v0.11.1
RUN go install github.com/sonalys/gotestfast/entrypoints/gotestfast@latest
RUN go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.3.0
RUN go install github.com/vektra/mockery/v2@v2.43.2
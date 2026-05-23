.PHONY: build install test fmt vet tidy

build:
	go build -o ssg ./cmd/ssg

install:
	go install ./cmd/ssg

test:
	go test ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

tidy:
	go mod tidy

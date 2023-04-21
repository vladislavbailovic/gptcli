GOFILES=$(shell find . -type f -name '*.go')

test:
	go test ./...

cover: test
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

.PHONY: build test check

build:
	go build -trimpath -o bin/unreal-agent-runner ./cmd/unreal-agent-runner

test:
	go test -race ./...

check:
	go vet ./...
	@test -z "$$(gofmt -l cmd harness internal)" || { gofmt -l cmd harness internal; exit 1; }

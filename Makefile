.PHONY: build
build:
	go build -o ./bin/app ./cmd/main.go
	chmod +x ./bin/app

lint:
	docker run --rm -e GOFLAGS='-buildvcs=false' -v $(shell pwd):/app -w /app 'golangci/golangci-lint:v1.64.4' sh -c \
    'golangci-lint run -v'

build_docker:
	buildx build . --tag dark705/go-ws-chat:1.0 --tag dark705/go-ws-chat:latest

run_docker:
	docker run  -p 8000:8000 dark705/go-ws-chat:latest
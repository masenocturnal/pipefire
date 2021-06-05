build:
	go build ./cmd/pipefired.go
build-release:
	go build -race -ldflags="-s -w" ./cmd/pipefired.go
run:
	go run -race ./cmd/pipefired.go
clean:
	rm -rf ./pipefired

start-containers:
	docker-compose -f docker/docker-compose.yml up


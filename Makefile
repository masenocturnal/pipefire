.PHONY: build build-release clean start-containers run

build:
	rm -rf ./build/plugins
	mkdir -p build/plugins
	go build  -buildmode=plugin -o ./build/plugins/directdebit.so ./pipelines/directdebit/directdebit.go 
	go build -o ./cmd/pipefired ./cmd/pipefired.go
	chmod +x ./cmd/pipefired

build-debug:
	rm -rf ./build/plugins
	mkdir -p build/plugins
	go build -gcflags='all=-N -l' -buildmode=plugin -o ./build/plugins/directdebit.so ./pipelines/directdebit/directdebit.go 
	go build -gcflags='all=-N -l' -o ./cmd/pipefired ./cmd/pipefired.go
	chmod +x ./cmd/pipefired
build-release:
	rm -rf ./build/
	mkdir -p build/plugins
	go build -race -ldflags="-s -w" -buildmode=plugin -o ./build/plugins/directdebit.so ./pipelines/directdebit/directdebit.go 
	go build -race -ldflags="-s -w"  -o ./build/pipefired ./cmd/pipefired.go
run:
	cd cmd; ./pipefired
clean:
	#rm -rf ./pipefired
	rm -rf build

start-containers:
	docker-compose -f docker/docker-compose.yml up


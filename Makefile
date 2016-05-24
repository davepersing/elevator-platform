.PHONY: default prebuild build run test clean

default: clean prebuild deps test build

PACKAGE_LIST := ./elevator ./elevator_service ./etcd ./http_api ./passenger ./scheduler ./util

test: prebuild
				go test ./...

prebuild: fmt vet

fmt:
	go fmt ./...

vet:
	go vet $(PACKAGE_LIST)

clean:
	rm -rf elevator-platform

deps:
	go get

build:
	go build

run:
	go run main.go $(PACKAGE_LIST)
